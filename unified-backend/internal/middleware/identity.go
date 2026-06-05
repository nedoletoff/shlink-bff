package middleware

import (
	"context"
	"net/http"
	"strings"
)

type ctxKey string

const (
	CtxKeySub      ctxKey = "sub"
	CtxKeyEmail    ctxKey = "email"
	CtxKeyUsername ctxKey = "username"
	CtxKeyRole     ctxKey = "role"
	CtxKeyGroups   ctxKey = "groups"

	// CtxKeyKeycloakRole — «основная» роль из Keycloak-групп (первое совпадение).
	// Используется для провизионирования и аудита.
	CtxKeyKeycloakRole ctxKey = "keycloak_role"

	// CtxKeyRoles — все роли пользователя, полученные из всех его Keycloak-групп.
	// Используется в RequirePermission для объединения прав (OR-семантика).
	CtxKeyRoles ctxKey = "roles"
)

// Identity — разобранный профиль из заголовков oauth2-proxy
type Identity struct {
	Sub          string
	Email        string
	Username     string
	Role         string   // основная роль (первое совпадение / из БД)
	Roles        []string // все роли из групп Keycloak текущего запроса
	KeycloakRole string   // основная роль из Keycloak-групп текущего запроса (всегда)
	Groups       []string
}

// ExtractIdentity читает X-Auth-Request-* заголовки и кладёт Identity в контекст.
//
// roleGroups — маппинг keycloak-group → role-name (из config.RoleGroups).
//
// Поле Role в контексте в режиме ROLE_SOURCE=keycloak заполняется прямо здесь.
// В режиме ROLE_SOURCE=db — здесь кладётся только keycloak_role;
// финальная роль (из БД) будет установлена в RequireActiveUser после загрузки user.
func ExtractIdentity(roleGroups map[string]string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sub := r.Header.Get("X-Auth-Request-User")
			if sub == "" {
				jsonError(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			groups := parseGroups(r.Header.Get("X-Auth-Request-Groups"))
			// Основная роль — первое совпадение (для провизионирования и БД).
			keycloakRole := resolveRole(groups, roleGroups)
			// Все роли — для объединения permissions.
			allRoles := resolveAllRoles(groups, roleGroups)

			ctx := context.WithValue(r.Context(), CtxKeySub, sub)
			ctx = context.WithValue(ctx, CtxKeyEmail, r.Header.Get("X-Auth-Request-Email"))
			ctx = context.WithValue(ctx, CtxKeyUsername, r.Header.Get("X-Auth-Request-Preferred-Username"))
			ctx = context.WithValue(ctx, CtxKeyGroups, groups)
			ctx = context.WithValue(ctx, CtxKeyRoles, allRoles)

			// CtxKeyKeycloakRole — сохраняем всегда для провизионирования и аудита.
			ctx = context.WithValue(ctx, CtxKeyKeycloakRole, keycloakRole)

			// CtxKeyRole — в режиме keycloak выставляем сразу.
			// В режиме db — пока пустая строка; RequireActiveUser перезапишет её из users.role.
			ctx = context.WithValue(ctx, CtxKeyRole, keycloakRole)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// IdentityFromCtx извлекает Identity из контекста запроса.
func IdentityFromCtx(ctx context.Context) *Identity {
	return &Identity{
		Sub:          strFromCtx(ctx, CtxKeySub),
		Email:        strFromCtx(ctx, CtxKeyEmail),
		Username:     strFromCtx(ctx, CtxKeyUsername),
		Role:         strFromCtx(ctx, CtxKeyRole),
		Roles:        rolesFromCtx(ctx),
		KeycloakRole: strFromCtx(ctx, CtxKeyKeycloakRole),
		Groups:       groupsFromCtx(ctx),
	}
}

func strFromCtx(ctx context.Context, k ctxKey) string {
	v, _ := ctx.Value(k).(string)
	return v
}

func groupsFromCtx(ctx context.Context) []string {
	v, _ := ctx.Value(CtxKeyGroups).([]string)
	return v
}

func rolesFromCtx(ctx context.Context) []string {
	v, _ := ctx.Value(CtxKeyRoles).([]string)
	return v
}

// resolveRole определяет основную роль пользователя по его группам Keycloak.
// Возвращает первое совпадение — используется для провизионирования и хранения в БД.
func resolveRole(groups []string, roleGroups map[string]string) string {
	for _, g := range groups {
		if role, ok := roleGroups[strings.ToLower(strings.TrimSpace(g))]; ok {
			return role
		}
	}
	return ""
}

// resolveAllRoles возвращает все уникальные роли пользователя из всех его групп.
// Порядок — по первому вхождению. Используется для объединения permissions (OR-семантика).
func resolveAllRoles(groups []string, roleGroups map[string]string) []string {
	seen := make(map[string]struct{}, len(groups))
	result := make([]string, 0, len(groups))
	for _, g := range groups {
		if role, ok := roleGroups[strings.ToLower(strings.TrimSpace(g))]; ok {
			if _, exists := seen[role]; !exists {
				seen[role] = struct{}{}
				result = append(result, role)
			}
		}
	}
	return result
}

// parseGroups: "group1,group2" → []string
func parseGroups(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			result = append(result, t)
		}
	}
	return result
}

// ClientIP возвращает IP клиента, доверяя заголовку X-Real-IP от nginx.
func ClientIP(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.IndexByte(xff, ','); idx >= 0 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	return r.RemoteAddr
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write([]byte(`{"error":"` + msg + `"}`))
}
