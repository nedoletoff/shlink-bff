package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"unified-backend/internal/domain"
	"unified-backend/internal/repository/postgres"
	"unified-backend/internal/service"
)

// AdminOnly разрешает доступ только роли с can_manage_users (или по совпадению с adminRole).
// Используется для /api/admin/* маршрутов.
func AdminOnly(adminRole string, auditRepo *postgres.AuditRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := UserFromCtx(r.Context())
			if user == nil || user.Role == "" {
				writeForbidden(w, r, user, auditRepo, "no identity")
				return
			}
			if user.Role != adminRole {
				writeForbidden(w, r, user, auditRepo, "role not admin")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequirePermission проверяет конкретный флаг permissions через PermissionsCache.
// Используется для гранулярного контроля отдельных эндпоинтов.
func RequirePermission(
	perms *service.PermissionsCache,
	check func(domain.RolePermissions) bool,
	auditRepo *postgres.AuditRepository,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := UserFromCtx(r.Context())
			if user == nil {
				writeForbidden(w, r, user, auditRepo, "no identity")
				return
			}
			p := perms.Get(user.Role)
			if !check(p) {
				writeForbidden(w, r, user, auditRepo, "permission denied")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole оставлен для обратной совместимости.
func RequireRole(role string, auditRepo *postgres.AuditRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := UserFromCtx(r.Context())
			if user == nil || user.Role != role {
				writeForbidden(w, r, user, auditRepo, "role mismatch")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeForbidden(w http.ResponseWriter, r *http.Request, user *domain.User, auditRepo *postgres.AuditRepository, reason string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte(`{"error":"forbidden"}`))

	if auditRepo == nil || user == nil {
		return
	}
	entry := &domain.AuditEntry{
		UserSub:   user.Sub,
		Username:  user.Username,
		Role:      string(user.Role),
		Action:    "access_denied",
		Resource:  r.URL.Path,
		Result:    "denied",
		Details:   map[string]any{"reason": reason},
		IPAddress: ClientIP(r),
		UserAgent: r.Header.Get("User-Agent"),
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		auditRepo.Record(ctx, entry)
	}()
}
