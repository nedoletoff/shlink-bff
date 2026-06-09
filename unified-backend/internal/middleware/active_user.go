package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"unified-backend/internal/config"
	"unified-backend/internal/domain"
	"unified-backend/internal/repository/postgres"
	"unified-backend/internal/shlinkctl"
)

// RequireActiveUser — middleware аутентификации и провизионирования.
// Проверяет только подлинность токена (через ExtractIdentity выше в цепочке)
// и статус пользователя (active/disabled). Проверка прав — исключительно через
// PermissionController в хендлерах или Authorize middleware.
func RequireActiveUser(
	userRepo *postgres.UserRepository,
	auditRepo *postgres.AuditRepository,
	provisioner *shlinkctl.Provisioner,
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

			// Единственная проверка авторизации здесь — статус active/disabled.
			// Все остальные проверки прав — через PermissionController.
			if user.Status == domain.StatusDisabled {
				slog.Warn("active_user: disabled user", "sub", user.Sub)
				writeErrJSON(w, http.StatusForbidden, "forbidden", "account is disabled")
				return
			}

			// Провизионируем shlink API-ключ при первом логине (идемпотентно).
			if user.ShlinkAPIKey == "" {
				key, provErr := provisioner.EnsureAPIKey(ctx, user.Sub, user.Username)
				if provErr != nil {
					slog.Warn("active_user: api key provisioning failed",
						"sub", user.Sub, "err", provErr)
				} else {
					user.ShlinkAPIKey = key
				}
			}

			ctx = WithUser(ctx, user)
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

	if user.Role != keycloakRole {
		slog.Info("active_user: role synced from keycloak",
			"sub", idnt.Sub, "old", user.Role, "new", keycloakRole)
		if err := repo.UpdateBySubFields(ctx, idnt.Sub, map[string]any{"role": keycloakRole}); err != nil {
			return nil, err
		}
		user.Role = keycloakRole
	}
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

	// Повторный визит — роль из БД, Keycloak не влияет.
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

func writeErrJSON(w http.ResponseWriter, code int, errMsg, reason string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	v := map[string]string{"error": errMsg}
	if reason != "" {
		v["reason"] = reason
	}
	_ = json.NewEncoder(w).Encode(v)
}
