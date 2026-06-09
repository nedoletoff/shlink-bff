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
	RoleSourceKeycloak RoleSource = "keycloak"

	// RoleSourceDB — роль читается из БД (users.role).
	RoleSourceDB RoleSource = "db"
)

type Config struct {
	// HTTPAddr — адрес для прослушивания (например ":8080").
	HTTPAddr string
	// Port — только номер порта, извлекается из HTTPAddr. Используется в http.Server.Addr.
	Port string

	DatabaseURL         string
	ShlinkURL           string
	// ShlinkBaseURL — алиас ShlinkURL для обратной совместимости.
	ShlinkBaseURL       string
	ShlinkDefaultDomain string

	// ShlinkAdminAPIKey — глобальный API-ключ администратора shlink.
	// Используется для вызовов PATCH /rest/v3/settings.
	// Читается из SHLINK_ADMIN_API_KEY.
	ShlinkAdminAPIKey string

	// ShlinkAPIKey — алиас ShlinkAdminAPIKey для обратной совместимости.
	// Deprecated: используйте ShlinkAdminAPIKey.
	ShlinkAPIKey string

	// CORSAllowedOrigins — список разрешённых origins (CORS_ALLOWED_ORIGINS, через запятую).
	// По умолчанию — ["*"].
	CORSAllowedOrigins []string

	// RoleGroups — маппинг keycloak-group (lower-case) → role-name.
	RoleGroups map[string]string

	// AdminRole — имя роли, считающейся администраторской.
	AdminRole string

	// RoleSource — источник истины для роли пользователя.
	RoleSource RoleSource

	// Feature flags
	UserSlugPrefixEnabled    bool
	UserTagInternalIdEnabled bool
	UserCustomSlugEnabled    bool

	// SHLINK_SHORT_ID_LENGTH=int (default: 0 = не передаём, shlink использует свой дефолт)
	ShlinkShortIDLength int

	// CLI provisioner
	ShlinkRunnerMode    string
	ShlinkContainerName string
	ShlinkBin           string

	// Ограничения по умолчанию для новых ссылок (0 = без ограничений).
	MaxVisitsDefault   int
	LinkTtlDefaultDays int
}

// MustLoad загружает конфиг и завершает процесс при отсутствии обязательных переменных.
func MustLoad() *Config {
	return Load()
}

func Load() *Config {
	httpAddr := getEnv("HTTP_ADDR", ":8080")
	shlinkURL := mustGetEnv("SHLINK_INTERNAL_URL")
	adminAPIKey := getEnv("SHLINK_ADMIN_API_KEY", "")
	cfg := &Config{
		HTTPAddr:                 httpAddr,
		Port:                     strings.TrimPrefix(httpAddr, ":"),
		DatabaseURL:              resolveDatabaseURL(),
		ShlinkURL:                shlinkURL,
		ShlinkBaseURL:            shlinkURL,
		ShlinkDefaultDomain:      getEnv("SHLINK_DEFAULT_DOMAIN", ""),
		ShlinkAdminAPIKey:        adminAPIKey,
		ShlinkAPIKey:             adminAPIKey, // alias
		CORSAllowedOrigins:       parseCORSOrigins(),
		RoleGroups:               parseRoleGroups(),
		AdminRole:                getEnv("ADMIN_ROLE", "admin"),
		RoleSource:               parseRoleSource(),
		UserSlugPrefixEnabled:    getBool("FEATURE_USER_SLUG_PREFIX", false),
		UserTagInternalIdEnabled: getBool("FEATURE_USER_TAG_INTERNAL_ID", false),
		UserCustomSlugEnabled:    getBool("FEATURE_USER_CUSTOM_SLUG", true),
		ShlinkShortIDLength:      getInt("SHLINK_SHORT_ID_LENGTH", 0),
		ShlinkRunnerMode:         getEnv("SHLINK_RUNNER_MODE", "docker"),
		ShlinkContainerName:      getEnv("SHLINK_CONTAINER", "shlink-api"),
		ShlinkBin:                getEnv("SHLINK_BIN", "shlink"),
		MaxVisitsDefault:         getInt("MAX_VISITS_DEFAULT", 0),
		LinkTtlDefaultDays:       getInt("LINK_TTL_DEFAULT_DAYS", 0),
	}
	slog.Info("config loaded",
		"role_source", cfg.RoleSource,
		"admin_role", cfg.AdminRole,
		"slug_prefix_enabled", cfg.UserSlugPrefixEnabled,
		"user_custom_slug_enabled", cfg.UserCustomSlugEnabled,
		"shlink_short_id_length", cfg.ShlinkShortIDLength,
		"shlink_default_domain", cfg.ShlinkDefaultDomain,
		"shlink_runner_mode", cfg.ShlinkRunnerMode,
		"cors_origins", cfg.CORSAllowedOrigins,
		"shlink_admin_api_key_set", cfg.ShlinkAdminAPIKey != "",
		"max_visits_default", cfg.MaxVisitsDefault,
		"link_ttl_default_days", cfg.LinkTtlDefaultDays,
	)
	return cfg
}

// parseCORSOrigins читает CORS_ALLOWED_ORIGINS (через запятую).
// По умолчанию возвращает ["*"].
func parseCORSOrigins() []string {
	raw := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if raw == "" {
		return []string{"*"}
	}
	var origins []string
	for _, o := range strings.Split(raw, ",") {
		if s := strings.TrimSpace(o); s != "" {
			origins = append(origins, s)
		}
	}
	if len(origins) == 0 {
		return []string{"*"}
	}
	return origins
}

// parseRoleSource читает ROLE_SOURCE. Допустимые значения: "keycloak", "db".
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

// resolveDatabaseURL строит DSN (приоритет по убыванию):
//  1. DATABASE_URL — если задан целиком.
//  2. DB_HOST + DB_PORT + DB_USER + DB_PASSWORD + DB_NAME + DB_SSLMODE.
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

func getInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
