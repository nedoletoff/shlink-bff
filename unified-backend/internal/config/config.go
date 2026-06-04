package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

// defaultRoleGroups — маппинг по умолчанию, если ROLE_GROUPS не задан.
const defaultRoleGroups = "shlink-admins=admin,admin=admin"

// RoleSource определяет, откуда берётся роль пользователя при каждом запросе.
type RoleSource string

const (
	// RoleSourceKeycloak — роль читается из X-Auth-Request-Groups на каждый запрос.
	// Keycloak — единственный источник истины. Изменение групп в Keycloak применяется сразу.
	// Роль в БД обновляется при каждом логине (upsert role).
	RoleSourceKeycloak RoleSource = "keycloak"

	// RoleSourceDB — роль читается из БД (users.role).
	// Первичный провизион берёт роль из Keycloak, далее роль управляется вручную через admin API.
	// Изменение групп в Keycloak НЕ влияет на роль уже существующего пользователя.
	RoleSourceDB RoleSource = "db"
)

type Config struct {
	HTTPAddr    string
	DatabaseURL string
	ShlinkURL   string

	// RoleGroups — маппинг keycloak-group (lower-case) → role-name.
	RoleGroups map[string]string

	// AdminRole — имя роли, считающейся администраторской.
	AdminRole string

	// RoleSource — источник истины для роли пользователя.
	// "keycloak" (default): роль берётся из Keycloak-групп на каждый запрос.
	// "db": роль берётся из users.role, Keycloak только для первичного провизионирования.
	RoleSource RoleSource

	// Feature flags
	UserSlugPrefixEnabled    bool
	UserTagInternalIdEnabled bool
}

func Load() *Config {
	cfg := &Config{
		HTTPAddr:                 getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:              resolveDatabaseURL(),
		ShlinkURL:                mustGetEnv("SHLINK_INTERNAL_URL"),
		RoleGroups:               parseRoleGroups(),
		AdminRole:                getEnv("ADMIN_ROLE", "admin"),
		RoleSource:               parseRoleSource(),
		UserSlugPrefixEnabled:    getBool("FEATURE_USER_SLUG_PREFIX", false),
		UserTagInternalIdEnabled: getBool("FEATURE_USER_TAG_INTERNAL_ID", false),
	}
	slog.Info("config loaded",
		"role_source", cfg.RoleSource,
		"admin_role", cfg.AdminRole,
		"slug_prefix_enabled", cfg.UserSlugPrefixEnabled,
	)
	return cfg
}

// parseRoleSource читает ROLE_SOURCE. Допустимые значения: "keycloak", "db".
// По умолчанию — "keycloak".
func parseRoleSource() RoleSource {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("ROLE_SOURCE")))
	switch RoleSource(v) {
	case RoleSourceDB:
		slog.Info("config: ROLE_SOURCE=db — role authority is the database")
		return RoleSourceDB
	case RoleSourceKeycloak, "":
		if v != "" && v != string(RoleSourceKeycloak) {
			slog.Warn("config: unknown ROLE_SOURCE value, defaulting to keycloak", "value", v)
		}
		return RoleSourceKeycloak
	default:
		slog.Warn("config: unknown ROLE_SOURCE value, defaulting to keycloak", "value", v)
		return RoleSourceKeycloak
	}
}

// resolveDatabaseURL строит DSN следующим образом (приоритет по убыванию):
//  1. DATABASE_URL — если задан целиком, используется как есть (legacy / внешний запуск).
//  2. DB_HOST + DB_PORT + DB_USER + DB_PASSWORD + DB_NAME + DB_SSLMODE — собирается DSN.
//     DB_NAME обязателен в этом режиме.
func resolveDatabaseURL() string {
	if url := strings.TrimSpace(os.Getenv("DATABASE_URL")); url != "" {
		return url
	}

	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := mustGetEnv("DB_USER")
	password := mustGetEnv("DB_PASSWORD")
	dbName := mustGetEnv("DB_NAME")
	sslMode := getEnv("DB_SSLMODE", "disable")

	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user, password, host, port, dbName, sslMode,
	)
}

// parseRoleGroups читает ROLE_GROUPS (приоритет) или ADMIN_GROUPS (обратная совместимость).
func parseRoleGroups() map[string]string {
	if raw := strings.TrimSpace(os.Getenv("ROLE_GROUPS")); raw != "" {
		return parseExplicitRoleGroups(raw)
	}

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
