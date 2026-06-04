package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"unified-backend/internal/config"
	"unified-backend/internal/domain"
	"unified-backend/internal/repository/postgres"
)

type userCtxKeyType struct{}

var userCtxKey = userCtxKeyType{}

// UserFromCtx возвращает *domain.User из контекста (устанавливается RequireActiveUser).
func UserFromCtx(ctx context.Context) *domain.User {
	v, _ := ctx.Value(userCtxKey).(*domain.User)
	return v
}

// RequireActiveUser — middleware провизионирования и авторизации.
//
// Логика зависит от cfg.RoleSource:
//
// ROLE_SOURCE=keycloak (default):
//   - Роль берётся из Keycloak-групп (CtxKeyKeycloakRole) при каждом запросе.
//   - Роль в БД ОБНОВЛЯЕТСЯ до текущей Keycloak-роли если изменилась.
//   - Keycloak — единственный источник истины.
//
// ROLE_SOURCE=db:
//   - Первый визит: роль берётся из Keycloak и записывается в БД (провизионирование).
//   - Последующие запросы: роль берётся из users.role (БД), Keycloak НЕ влияет.
//   - Смена роли — только через admin API (PUT /api/admin/users/{sub}).
func RequireActiveUser(
	userRepo *postgres.UserRepository,
	auditRepo *postgres.AuditRepository,
	cfg *config.Config,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			idnt := IdentityFromCtx(ctx)

			if idnt.Sub == "" {
				writeErrJSON(w, http.StatusUnauthorized, "unauthorized", "")
				return
			}

			keycloakRole := idnt.KeycloakRole
			if keycloakRole == "" {
				slog.Warn("active_user: no known keycloak group", "sub", idnt.Sub)
				writeErrJSON(w, http.StatusForbidden, "forbidden", "user is not a member of any configured group")
				return
			}

			user, err := userRepo.GetBySub(ctx, idnt.Sub)
			if err != nil {
				slog.Error("active_user: db lookup failed", "sub", idnt.Sub, "err", err)
				writeErrJSON(w, http.StatusInternalServerError, "internal server error", "")
				return
			}

			switch cfg.RoleSource {
			case config.RoleSourceDB:
				user, err = handleRoleSourceDB(ctx, userRepo, user, idnt, keycloakRole)
			default: // RoleSourceKeycloak
				user, err = handleRoleSourceKeycloak(ctx, userRepo, user, idnt, keycloakRole)
			}

			if err != nil {
				slog.Error("active_user: provisioning failed", "sub", idnt.Sub, "err", err)
				writeErrJSON(w, http.StatusInternalServerError, "internal server error", "")
				return
			}

			if user.Status == domain.StatusDisabled {
				slog.Warn("active_user: disabled user", "sub", user.Sub)
				writeErrJSON(w, http.StatusForbidden, "forbidden", "account is disabled")
				return
			}

			// Кладём пользователя и финальную роль в контекст.
			ctx = context.WithValue(ctx, userCtxKey, user)
			ctx = context.WithValue(ctx, CtxKeyRole, user.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// handleRoleSourceKeycloak — ROLE_SOURCE=keycloak (default).
func handleRoleSourceKeycloak(
	ctx context.Context,
	repo *postgres.UserRepository,
	user *domain.User,
	idnt *Identity,
	keycloakRole string,
) (*domain.User, error) {
	if user == nil {
		newUser := &domain.User{
			Sub:      idnt.Sub,
			Email:    idnt.Email,
			Username: idnt.Username,
			Role:     keycloakRole,
			Status:   domain.StatusActive,
		}
		if err := repo.Upsert(ctx, newUser); err != nil {
			return nil, err
		}
		slog.Info("active_user: provisioned", "sub", idnt.Sub, "role", keycloakRole, "source", "keycloak")
		return newUser, nil
	}

	// Роль обновляем если изменилась в Keycloak.
	if user.Role != keycloakRole {
		slog.Info("active_user: role synced from keycloak",
			"sub", idnt.Sub, "old", user.Role, "new", keycloakRole)
		if err := repo.UpdateBySubFields(ctx, idnt.Sub, map[string]any{"role": keycloakRole}); err != nil {
			return nil, err
		}
		user.Role = keycloakRole
	}
	// username/email всегда синхронизируем из Keycloak.
	fields := map[string]any{}
	if user.Username != idnt.Username {
		fields["username"] = idnt.Username
		user.Username = idnt.Username
	}
	if user.Email != idnt.Email {
		fields["email"] = idnt.Email
		user.Email = idnt.Email
	}
	if len(fields) > 0 {
		if err := repo.UpdateBySubFields(ctx, idnt.Sub, fields); err != nil {
			return nil, err
		}
	}
	return user, nil
}

// handleRoleSourceDB — ROLE_SOURCE=db.
func handleRoleSourceDB(
	ctx context.Context,
	repo *postgres.UserRepository,
	user *domain.User,
	idnt *Identity,
	keycloakRole string,
) (*domain.User, error) {
	if user == nil {
		// Только при первом визите роль берётся из Keycloak.
		newUser := &domain.User{
			Sub:      idnt.Sub,
			Email:    idnt.Email,
			Username: idnt.Username,
			Role:     keycloakRole,
			Status:   domain.StatusActive,
		}
		if err := repo.Upsert(ctx, newUser); err != nil {
			return nil, err
		}
		slog.Info("active_user: provisioned", "sub", idnt.Sub, "role", keycloakRole, "source", "db (first visit)")
		return newUser, nil
	}

	// Повторный визит — роль из БД, Keycloak не меняет её.
	// username/email обновляем (они не секретны и полезны для аудита).
	fields := map[string]any{}
	if user.Username != idnt.Username {
		fields["username"] = idnt.Username
		user.Username = idnt.Username
	}
	if user.Email != idnt.Email {
		fields["email"] = idnt.Email
		user.Email = idnt.Email
	}
	if len(fields) > 0 {
		if err := repo.UpdateBySubFields(ctx, idnt.Sub, fields); err != nil {
			return nil, err
		}
	}
	// user.Role — из БД. Keycloak-роль НЕ применяется.
	return user, nil
}

func writeErrJSON(w http.ResponseWriter, code int, errMsg, reason string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	v := map[string]string{"error": errMsg}
	if reason != "" {
		v["reason"] = reason
	}
	_ = json.NewEncoder(w).Encode(v)
}
