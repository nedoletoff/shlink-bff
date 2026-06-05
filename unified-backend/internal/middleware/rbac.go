package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"unified-backend/internal/domain"
	"unified-backend/internal/repository/postgres"
	"unified-backend/internal/service"
)

// AdminOnly разрешает доступ только роли с can_manage_users (или по совпадению с adminRole).
// Используется для /api/admin/* маршрутов.
// Сравнение регистронезависимое — "Admin" == "admin".
func AdminOnly(adminRole string, auditRepo *postgres.AuditRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := UserFromCtx(r.Context())
			if user == nil || user.Role == "" {
				writeForbidden(w, r, user, auditRepo, "no identity")
				return
			}
			if !strings.EqualFold(string(user.Role), adminRole) {
				writeForbidden(w, r, user, auditRepo, "role not admin")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequirePermission проверяет конкретный флаг permissions с учётом всех ролей пользователя.
// Использует OR-семантику: разрешение выдаётся, если хотя бы одна роль его имеет.
// Роли берутся из CtxKeyRoles (все Keycloak-группы) или fallback на user.Role из БД.
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

			// Собираем все роли: из контекста (Keycloak) + роль из БД (если не пересекается).
			roles := effectiveRoles(r.Context(), user.Role)
			p := perms.GetMerged(roles)
			if !check(p) {
				writeForbidden(w, r, user, auditRepo, "permission denied")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// effectiveRoles возвращает объединённый список ролей для проверки permissions.
// Приоритет: все роли из Keycloak-групп (CtxKeyRoles), дополненные ролью из БД.
func effectiveRoles(ctx context.Context, dbRole string) []string {
	keycloakRoles := rolesFromCtx(ctx) // все роли из групп Keycloak
	if len(keycloakRoles) == 0 {
		// Fallback: только роль из БД (ROLE_SOURCE=db или заголовки отсутствуют)
		if dbRole != "" {
			return []string{dbRole}
		}
		return nil
	}
	// Добавляем dbRole если она не входит в keycloakRoles (ROLE_SOURCE=db)
	seen := make(map[string]struct{}, len(keycloakRoles))
	for _, r := range keycloakRoles {
		seen[strings.ToLower(r)] = struct{}{}
	}
	if dbRole != "" {
		if _, ok := seen[strings.ToLower(dbRole)]; !ok {
			return append(keycloakRoles, dbRole)
		}
	}
	return keycloakRoles
}

// RequireRole оставлен для обратной совместимости.
// Сравнение регистронезависимое.
func RequireRole(role string, auditRepo *postgres.AuditRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := UserFromCtx(r.Context())
			if user == nil || !strings.EqualFold(string(user.Role), role) {
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
