package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
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

	// CtxKeyKeycloakRole — основная роль из Keycloak-групп (первое совпадение)
	CtxKeyKeycloakRole ctxKey = "keycloak_role"

	// CtxKeyRoles — все роли пользователя из всех его Keycloak-групп
	CtxKeyRoles ctxKey = "roles"
)

type Identity struct {
	Sub          string
	Email        string
	Username     string
	Role         string   // финальная роль (из БД или keycloak)
	Roles        []string // все роли из групп Keycloak
	KeycloakRole string
	Groups       []string
}

// ExtractIdentity читает X-Auth-Request-* заголовки и кладёт Identity в контекст.
func ExtractIdentity(roleGroups map[string]string, trustedSecret ...string) func(http.Handler) http.Handler {
	secret := ""
	if len(trustedSecret) > 0 {
		secret = trustedSecret[0]
	}
	if secret == "" {
		slog.Warn("identity: TRUSTED_HEADER_SECRET is not set — header trust is unrestricted; set the env var in production")
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sub := r.Header.Get("X-Auth-Request-User")
			if sub == "" {
				jsonError(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			if secret != "" {
				sig := r.Header.Get("X-Auth-Signature")
				if !validateHMAC(sub, secret, sig) {
					slog.Warn("identity: invalid X-Auth-Signature", "sub", sub, "ip", ClientIP(r))
					jsonError(w, "unauthorized", http.StatusUnauthorized)
					return
				}
			}

			groups := parseGroups(r.Header.Get("X-Auth-Request-Groups"))
			keycloakRole := resolveRole(groups, roleGroups)
			allRoles := resolveAllRoles(groups, roleGroups)

			ctx := context.WithValue(r.Context(), CtxKeySub, sub)
			ctx = context.WithValue(ctx, CtxKeyEmail, r.Header.Get("X-Auth-Request-Email"))
			ctx = context.WithValue(ctx, CtxKeyUsername, r.Header.Get("X-Auth-Request-Preferred-Username"))
			ctx = context.WithValue(ctx, CtxKeyGroups, groups)
			ctx = context.WithValue(ctx, CtxKeyRoles, allRoles)
			ctx = context.WithValue(ctx, CtxKeyKeycloakRole, keycloakRole)
			ctx = context.WithValue(ctx, CtxKeyRole, keycloakRole) // временно, потом перезапишется

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func validateHMAC(message, secret, sig string) bool {
	if sig == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	expected := hex.EncodeToString(mac.Sum(nil))
	expectedBytes, err1 := hex.DecodeString(expected)
	gotBytes, err2 := hex.DecodeString(sig)
	if err1 != nil || err2 != nil {
		return false
	}
	return hmac.Equal(expectedBytes, gotBytes)
}

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

func resolveRole(groups []string, roleGroups map[string]string, defaultRole ...string) string {
	for _, g := range groups {
		if role, ok := roleGroups[strings.ToLower(strings.TrimSpace(g))]; ok {
			return role
		}
	}
	if len(defaultRole) > 0 && defaultRole[0] != "" {
		return defaultRole[0]
	}
	return ""
}

func resolveAllRoles(groups []string, roleGroups map[string]string, defaultRole ...string) []string {
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
	if len(result) == 0 && len(defaultRole) > 0 && defaultRole[0] != "" {
		return []string{defaultRole[0]}
	}
	return result
}

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

