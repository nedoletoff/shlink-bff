package middleware

import (
	"log/slog"
	"net/http"

	"unified-backend/internal/domain"
	"unified-backend/internal/repository/postgres"
)

// RequireRole возвращает middleware, проверяющий роль пользователя.
// При нарушении: 403 + асинхронная запись в аудит.
func RequireRole(role domain.Role, auditRepo *postgres.AuditRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := IdentityFromCtx(r.Context())
			if domain.Role(id.Role) != role {
				slog.Warn("rbac: access denied",
					"sub",      id.Sub,
					"username", id.Username,
					"role",     id.Role,
					"required", string(role),
					"path",     r.URL.Path,
					"method",   r.Method,
				)
				go auditRepo.Record(r.Context(), &domain.AuditEntry{
					UserSub:  id.Sub,
					Username: id.Username,
					Role:     id.Role,
					Action:   "rbac_denied",
					Resource: r.URL.Path,
					Result:   "denied",
					Details:  map[string]any{"method": r.Method, "required_role": string(role)},
				})
				jsonError(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// AdminOnly — сокращение для RequireRole("admin")
func AdminOnly(auditRepo *postgres.AuditRepository) func(http.Handler) http.Handler {
	return RequireRole(domain.RoleAdmin, auditRepo)
}

// RequireActiveUser загружает пользователя из БД, проверяет статус active
// и кладёт *domain.User в контекст для последующих хендлеров.
func RequireActiveUser(userRepo *postgres.UserRepository, auditRepo *postgres.AuditRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := IdentityFromCtx(r.Context())

			user, err := userRepo.GetBySub(r.Context(), id.Sub)
			if err != nil {
				slog.Error("rbac: db error on user lookup", "sub", id.Sub, "err", err)
				jsonError(w, "internal error", http.StatusInternalServerError)
				return
			}
			if user == nil || user.Status != domain.StatusActive {
				jsonError(w, "forbidden: user not provisioned or inactive", http.StatusForbidden)
				return
			}

			ctx := WithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
