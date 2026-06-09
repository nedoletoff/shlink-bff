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

	"github.com/google/uuid"
)

type PermInvalidator interface {
	InvalidateUser(userID uuid.UUID)
}

// RequireActiveUser — middleware аутентификации и провизионирования.
func RequireActiveUser(
	userRepo *postgres.UserRepository,
	auditRepo *postgres.AuditRepository,
	provisioner *shlinkctl.Provisioner,
	cfg *config.Config,
	permInvalidator ...PermInvalidator,
) func(http.Handler) http.Handler {
	var inv PermInvalidator
	if len(permInvalidator) > 0 {
		inv = permInvalidator[0]
	}

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
			default:
				user, err = handleRoleSourceKeycloak(ctx, userRepo, user, idnt, keycloakRole, inv)
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

			if user.ShlinkAPIKey == "" {
				key, provErr := provisioner.EnsureAPIKey(ctx, user.Sub, user.Username)
				if provErr != nil {
					slog.Warn("active_user: api key provisioning failed", "sub", user.Sub, "err", provErr)
				} else {
					user.ShlinkAPIKey = key
				}
			}

			// Кладём пользователя в контекст
			ctx = WithUser(ctx, user)
			// Обновляем role в контексте (используем user.Role, который уже содержит актуальное имя)
			ctx = context.WithValue(ctx, CtxKeyRole, user.Role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// handleRoleSourceKeycloak — синхронизация роли из Keycloak
func handleRoleSourceKeycloak(
	ctx context.Context,
	repo *postgres.UserRepository,
	user *domain.User,
	idnt *Identity,
	keycloakRole string,
	inv PermInvalidator,
) (*domain.User, error) {
	if user == nil {
		// Создаём нового пользователя
		newUser := &domain.User{
			Sub:      idnt.Sub,
			Email:    idnt.Email,
			Username: idnt.Username,
			Role:     keycloakRole, // role_text
			Status:   domain.StatusActive,
		}
		if err := repo.Upsert(ctx, newUser); err != nil {
			return nil, err
		}
		// Синхронизируем role_id
		if err := syncRoleID(ctx, repo, newUser, keycloakRole, inv); err != nil {
			slog.Warn("active_user: role_id sync failed on provision", "sub", idnt.Sub, "err", err)
		}
		slog.Info("active_user: provisioned", "sub", idnt.Sub, "role", keycloakRole)
		return newUser, nil
	}

	fields := map[string]any{}

	if user.Role != keycloakRole {
		slog.Info("active_user: role synced from keycloak", "sub", idnt.Sub, "old", user.Role, "new", keycloakRole)
		fields["role_text"] = keycloakRole
		user.Role = keycloakRole
	}
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

	// Если role_id не заполнен или роль изменилась – синхронизируем
	if user.RoleID == nil || (fields["role_text"] != nil) {
		if err := syncRoleID(ctx, repo, user, keycloakRole, inv); err != nil {
			slog.Warn("active_user: role_id sync failed", "sub", idnt.Sub, "err", err)
		}
	}

	return user, nil
}

// syncRoleID обновляет users.role_id по имени роли и инвалидирует кэш
func syncRoleID(ctx context.Context, repo *postgres.UserRepository, user *domain.User, roleName string, inv PermInvalidator) error {
	roleID, err := repo.GetRoleIDByName(ctx, roleName)
	if err != nil {
		return err
	}
	if roleID == nil {
		return nil // роль ещё не заведена в БД – игнорируем
	}
	if user.RoleID != nil && *user.RoleID == *roleID {
		return nil
	}
	if err := repo.UpdateBySubFields(ctx, user.Sub, map[string]any{"role_id": roleID}); err != nil {
		return err
	}
	user.RoleID = roleID
	if inv != nil {
		inv.InvalidateUser(user.ID)
	}
	return nil
}

// handleRoleSourceDB — роль всегда из БД, только обновляем username/email
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
		slog.Info("active_user: provisioned", "sub", idnt.Sub, "role", keycloakRole, "source", "db")
		return newUser, nil
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

func writeErrJSON(w http.ResponseWriter, code int, errMsg, reason string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	v := map[string]string{"error": errMsg}
	if reason != "" {
		v["reason"] = reason
	}
	_ = json.NewEncoder(w).Encode(v)
}

