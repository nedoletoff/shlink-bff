package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"
)

// defaultRoleGroups — маппинг по умолчанию, если ROLE_GROUPS не задан.
// Формат: keycloak-group=role-name,...
// Пример: "shlink-admin=shlink-admin,shlink-user=shlink-user"
//
// Обратная совместимость: если задан только ADMIN_GROUPS (старый формат),
// он интерпретируется как список групп, получающих роль "admin".
const defaultRoleGroups = "shlink-admins=admin,admin=admin"

type Config struct {
	HTTPAddr    string
	DatabaseURL string
	ShlinkURL   string

	// RoleGroups — маппинг keycloak-group (lower-case) → role-name.
	// Читается один раз при старте из ROLE_GROUPS (формат: group=role,...).
	// Иммутабельно после загрузки — гонок данных нет.
	RoleGroups map[string]string

	// AdminRole — имя роли, считающейся администраторской (для RBAC-проверок AdminOnly).
	// Задаётся через ADMIN_ROLE, по умолчанию "admin".
	AdminRole string

	// Feature flags
	UserSlugPrefixEnabled    bool
	UserTagInternalIdEnabled bool
}

func Load() *Config {
	cfg := &Config{
		HTTPAddr:                 getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:              mustGetEnv("DATABASE_URL"),
		ShlinkURL:                mustGetEnv("SHLINK_INTERNAL_URL"),
		RoleGroups:               parseRoleGroups(),
		AdminRole:                getEnv("ADMIN_ROLE", "admin"),
		UserSlugPrefixEnabled:    getBool("FEATURE_USER_SLUG_PREFIX", false),
		UserTagInternalIdEnabled: getBool("FEATURE_USER_TAG_INTERNAL_ID", false),
	}
	return cfg
}

// parseRoleGroups читает ROLE_GROUPS (приоритет) или ADMIN_GROUPS (обратная совместимость).
//
// ROLE_GROUPS format: "keycloak-group=role-name,..." — явный маппинг группа→роль.
// ADMIN_GROUPS format: "group1,group2,..." — все перечисленные группы получают ADMIN_ROLE.
func parseRoleGroups() map[string]string {
	if raw := strings.TrimSpace(os.Getenv("ROLE_GROUPS")); raw != "" {
		return parseExplicitRoleGroups(raw)
	}

	// Обратная совместимость: ADMIN_GROUPS
	adminRole := getEnv("ADMIN_ROLE", "admin")
	if raw := strings.TrimSpace(os.Getenv("ADMIN_GROUPS")); raw != "" {
		m := make(map[string]string)
		for _, g := range strings.Split(raw, ",") {
			if t := strings.ToLower(strings.TrimSpace(g)); t != "" {
				m[t] = adminRole
			}
		}
		slog.Warn("config: ADMIN_GROUPS is deprecated, use ROLE_GROUPS instead")
		return m
	}

	return parseExplicitRoleGroups(defaultRoleGroups)
}

func parseExplicitRoleGroups(raw string) map[string]string {
	m := make(map[string]string)
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		idx := strings.IndexByte(entry, '=')
		if idx < 0 {
			slog.Warn("config: ROLE_GROUPS entry missing '=', skipping", "entry", entry)
			continue
		}
		group := strings.ToLower(strings.TrimSpace(entry[:idx]))
		role := strings.TrimSpace(entry[idx+1:])
		if group != "" && role != "" {
			m[group] = role
		}
	}
	return m
}

// ParseAdminGroups оставлен для обратной совместимости с тестами.
// Deprecated: используйте parseRoleGroups.
func ParseAdminGroups(raw string) map[string]struct{} {
	if strings.TrimSpace(raw) == "" {
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

func mustGetEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.Error("required env variable is missing", "key", key)
		os.Exit(1)
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}
