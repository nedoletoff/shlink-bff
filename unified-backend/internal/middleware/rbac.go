package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"unified-backend/internal/domain"
	"unified-backend/internal/repository/postgres"
)

// recordAuditAsync пишет запись аудита в горутине с ДЕТАЧНУТЫМ контекстом.
// r.Context() отменяется после возврата handler, из-за чего запись терялась (#10).
func recordAuditAsync(auditRepo *postgres.AuditRepository, entry *domain.AuditEntry) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		auditRepo.Record(ctx, entry)
	}()
}

// RequireRole возвращает middleware, проверяющий роль пользователя.
// При нарушении: 403 + асинхронная запись в аудит.
func RequireRole(role domain.Role, auditRepo *postgres.AuditRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := IdentityFromCtx(r.Context())
			if domain.Role(id.Role) != role {
				slog.Warn("rbac: access denied",
					"sub", id.Sub,
					"username", id.Username,
					"role", id.Role,
					"required", string(role),
					"path", r.URL.Path,
					"method", r.Method,
				)
				recordAuditAsync(auditRepo, &domain.AuditEntry{
					UserSub:   id.Sub,
					Username:  id.Username,
					Role:      id.Role,
					Action:    "rbac_denied",
					Resource:  r.URL.Path,
					Result:    "denied",
					Details:   map[string]any{"method": r.Method, "required_role": string(role)},
					IPAddress: r.RemoteAddr,
					UserAgent: r.Header.Get("User-Agent"),
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
//
// Auto-provision: если пользователь впервые логинится (нет записи в БД),
// создаём её автоматически на основе заголовков oauth2-proxy.
// Роль берётся из resolveRole (ADMIN_GROUPS), статус = active.
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

			// Первый логин — провизионируем пользователя автоматически
			if user == nil {
				role := domain.Role(id.Role)
				if role != domain.RoleAdmin && role != domain.RoleUser {
					role = domain.RoleUser
				}
				newUser := &domain.User{
					Sub:      id.Sub,
					Username: id.Username,
					Email:    id.Email,
					Role:     role,
					Status:   domain.StatusActive,
				}
				if upsertErr := userRepo.Upsert(r.Context(), newUser); upsertErr != nil {
					slog.Error("rbac: failed to auto-provision user", "sub", id.Sub, "err", upsertErr)
					jsonError(w, "internal error", http.StatusInternalServerError)
					return
				}
				slog.Info("rbac: auto-provisioned new user", "sub", id.Sub, "username", id.Username, "role", string(role))
				// Перечитываем, чтобы получить id и created_at из БД
				user, err = userRepo.GetBySub(r.Context(), id.Sub)
				if err != nil || user == nil {
					slog.Error("rbac: failed to reload user after provision", "sub", id.Sub, "err", err)
					jsonError(w, "internal error", http.StatusInternalServerError)
					return
				}
			}

			if user.Status != domain.StatusActive {
				jsonError(w, "forbidden: user inactive", http.StatusForbidden)
				return
			}

			ctx := WithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
