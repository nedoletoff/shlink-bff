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
)

// Identity — разобранный профиль из заголовков oauth2-proxy
type Identity struct {
	Sub      string
	Email    string
	Username string
	Role     string
	Groups   []string
}

// ExtractIdentity — фабрика middleware: читает X-Auth-Request-* заголовки от oauth2-proxy
// и кладёт Identity-поля в контекст. Множество admin-групп передаётся явно
// (из config.Config), без глобального мутабельного состояния — поэтому гонок данных нет (#13, #33).
func ExtractIdentity(adminGroups map[string]struct{}) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sub := r.Header.Get("X-Auth-Request-User")
			if sub == "" {
				jsonError(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			groups := parseGroups(r.Header.Get("X-Auth-Request-Groups"))
			role := resolveRole(groups, adminGroups)

			ctx := context.WithValue(r.Context(), CtxKeySub, sub)
			ctx = context.WithValue(ctx, CtxKeyEmail, r.Header.Get("X-Auth-Request-Email"))
			ctx = context.WithValue(ctx, CtxKeyUsername, r.Header.Get("X-Auth-Request-Preferred-Username"))
			ctx = context.WithValue(ctx, CtxKeyRole, role)
			ctx = context.WithValue(ctx, CtxKeyGroups, groups)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// IdentityFromCtx извлекает Identity из контекста запроса
func IdentityFromCtx(ctx context.Context) *Identity {
	return &Identity{
		Sub:      strFromCtx(ctx, CtxKeySub),
		Email:    strFromCtx(ctx, CtxKeyEmail),
		Username: strFromCtx(ctx, CtxKeyUsername),
		Role:     strFromCtx(ctx, CtxKeyRole),
		Groups:   groupsFromCtx(ctx),
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

// resolveRole проверяет группы пользователя против переданного множества adminGroups.
// Сравнение case-insensitive. Если совпадение найдено — возвращает "admin", иначе "user".
func resolveRole(groups []string, adminGroups map[string]struct{}) string {
	for _, g := range groups {
		if _, ok := adminGroups[strings.ToLower(strings.TrimSpace(g))]; ok {
			return "admin"
		}
	}
	return "user"
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
//
// Мы намеренно не используем chi RealIP (уязвим к IP-spoofing). nginx — единственная
// точка входа и выставляет X-Real-IP из $remote_addr, поэтому ему можно доверять.
// Fallback — r.RemoteAddr (прямое подключение без прокси, например в тестах).
func ClientIP(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Первый в списке — исходный клиент.
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
