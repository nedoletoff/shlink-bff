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
// Приоритет: все роли из Keycloak-групп (СтхКеуРолес), дополненные ролью из БД.
func effectiveRoles(ctx context.Context, dbRole string) []string {
	keycloakRoles := rolesFromCtx(ctx)
	if len(keycloakRoles) == 0 {
		if dbRole != "" {
			return []string{dbRole}
		}
		return nil
	}
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

// RolesFromCtx returns effective roles from context and DB role fallback.
func RolesFromCtx(ctx context.Context, dbRole string) []string {
	return effectiveRoles(ctx, dbRole)
}
