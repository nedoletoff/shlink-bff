package middleware

import (
	"context"
	"net/http"
	"os"
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

// adminGroups — множество имён групп, которым присваивается роль admin.
// Загружается из переменной окружения ADMIN_GROUPS (comma-separated).
// Дефолт: "shlink-admins,admin"
var adminGroups = loadAdminGroups()

func loadAdminGroups() map[string]struct{} {
	raw := os.Getenv("ADMIN_GROUPS")
	if raw == "" {
		raw = "shlink-admins,admin"
	}
	m := make(map[string]struct{})
	for _, g := range strings.Split(raw, ",") {
		if t := strings.ToLower(strings.TrimSpace(g)); t != "" {
			m[t] = struct{}{}
		}
	}
	return m
}

// ReloadAdminGroups перечитывает ADMIN_GROUPS из env.
// Используется только в тестах для изоляции состояния между кейсами.
func ReloadAdminGroups() {
	adminGroups = loadAdminGroups()
}

// ExtractIdentity читает X-Auth-Request-* заголовки от oauth2-proxy
// и кладёт Identity-поля в контекст.
func ExtractIdentity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sub := r.Header.Get("X-Auth-Request-User")
		if sub == "" {
			jsonError(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		groups := parseGroups(r.Header.Get("X-Auth-Request-Groups"))
		role := resolveRole(groups)

		ctx := context.WithValue(r.Context(), CtxKeySub, sub)
		ctx = context.WithValue(ctx, CtxKeyEmail, r.Header.Get("X-Auth-Request-Email"))
		ctx = context.WithValue(ctx, CtxKeyUsername, r.Header.Get("X-Auth-Request-Preferred-Username"))
		ctx = context.WithValue(ctx, CtxKeyRole, role)
		ctx = context.WithValue(ctx, CtxKeyGroups, groups)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
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

// resolveRole проверяет группы пользователя против adminGroups.
// Сравнение case-insensitive. Если совпадение найдено — возвращает "admin", иначе "user".
func resolveRole(groups []string) string {
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

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write([]byte(`{"error":"` + msg + `"}`))
}
