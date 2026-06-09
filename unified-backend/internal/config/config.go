package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

const defaultRoleGroups = "shlink-admins=admin,admin=admin"

type RoleSource string

const (
	RoleSourceKeycloak RoleSource = "keycloak"
	RoleSourceDB       RoleSource = "db"
)

type Config struct {
	HTTPAddr                 string
	Port                     string
	DatabaseURL              string
	ShlinkInternalURL        string
	ShlinkDefaultDomain      string
	ShlinkAdminAPIKey        string
	CORSAllowedOrigins       []string
	CORSAllowedMethods       []string
	CORSAllowedHeaders       []string
	TrustedHeaderSecret      string
	DefaultRole              string
	RoleGroups               map[string]string
	AdminRole                string
	RoleSource               RoleSource
	UserSlugPrefixEnabled    bool
	UserTagInternalIdEnabled bool
	UserCustomSlugEnabled    bool
	ShlinkShortIDLength      int
	ShlinkRunnerMode         string
	ShlinkContainerName      string
	ShlinkBin                string
	MaxVisitsDefault         int
	LinkTtlDefaultDays       int
	BulkOperationLimit       int
}

func MustLoad() *Config {
	return Load()
}

func Load() *Config {
	httpAddr := getEnv("HTTP_ADDR", ":8080")

	cfg := &Config{
		HTTPAddr:                 httpAddr,
		Port:                     strings.TrimPrefix(httpAddr, ":"),
		DatabaseURL:              resolveDatabaseURL(),
		ShlinkInternalURL:        mustGetEnv("SHLINK_INTERNAL_URL"),
		ShlinkDefaultDomain:      getEnv("SHLINK_DEFAULT_DOMAIN", ""),
		ShlinkAdminAPIKey:        getEnv("SHLINK_ADMIN_API_KEY", ""),
		CORSAllowedOrigins:       parseCORSOrigins(),
		CORSAllowedMethods:       parseCORSMethods(),
		CORSAllowedHeaders:       parseCORSHeaders(),
		TrustedHeaderSecret:      getEnv("TRUSTED_HEADER_SECRET", ""),
		DefaultRole:              getEnv("DEFAULT_ROLE", ""),
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
		BulkOperationLimit:       getInt("BULK_OPERATION_LIMIT", 100),
	}

	slog.Info(
		"config loaded",
		"http_addr", cfg.HTTPAddr,
		"database_url", maskDSN(cfg.DatabaseURL),
		"shlink_internal_url", cfg.ShlinkInternalURL,
		"shlink_admin_api_key_set", cfg.ShlinkAdminAPIKey != "",
		"role_source", cfg.RoleSource,
		"admin_role", cfg.AdminRole,
	)
	return cfg
}

func parseCORSOrigins() []string {
	raw := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if raw == "" {
		return []string{"*"}
	}
	var origins []string
	for part := range strings.SplitSeq(raw, ",") {
		if s := strings.TrimSpace(part); s != "" {
			origins = append(origins, s)
		}
	}
	if len(origins) == 0 {
		return []string{"*"}
	}
	return origins
}

func parseCORSMethods() []string {
	raw := strings.TrimSpace(os.Getenv("CORS_ALLOWED_METHODS"))
	if raw == "" {
		return []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	}
	var methods []string
	for part := range strings.SplitSeq(raw, ",") {
		if s := strings.ToUpper(strings.TrimSpace(part)); s != "" {
			methods = append(methods, s)
		}
	}
	if len(methods) == 0 {
		return []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	}
	return methods
}

func parseCORSHeaders() []string {
	raw := strings.TrimSpace(os.Getenv("CORS_ALLOWED_HEADERS"))
	if raw == "" {
		return []string{"Authorization", "Content-Type"}
	}
	var headers []string
	for part := range strings.SplitSeq(raw, ",") {
		if s := strings.TrimSpace(part); s != "" {
			headers = append(headers, s)
		}
	}
	if len(headers) == 0 {
		return []string{"Authorization", "Content-Type"}
	}
	return headers
}

func parseRoleSource() RoleSource {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("ROLE_SOURCE")))
	switch RoleSource(v) {
	case RoleSourceDB:
		return RoleSourceDB
	case RoleSourceKeycloak, "":
		if v != "" && v != string(RoleSourceKeycloak) {
			slog.Warn("unknown ROLE_SOURCE, fallback to keycloak", "value", v)
		}
		return RoleSourceKeycloak
	default:
		return RoleSourceKeycloak
	}
}

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
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, password, host, port, dbName, sslMode)
}

func parseRoleGroups() map[string]string {
	if raw := strings.TrimSpace(os.Getenv("ROLE_GROUPS")); raw != "" {
		return parseExplicitRoleGroups(raw)
	}
	adminRole := getEnv("ADMIN_ROLE", "admin")
	if raw := strings.TrimSpace(os.Getenv("ADMIN_GROUPS")); raw != "" {
		m := make(map[string]string)
		for g := range strings.SplitSeq(raw, ",") {
			if t := strings.ToLower(strings.TrimSpace(g)); t != "" {
				m[t] = adminRole
			}
		}
		slog.Warn("ADMIN_GROUPS is deprecated, use ROLE_GROUPS")
		return m
	}
	return parseExplicitRoleGroups(defaultRoleGroups)
}

func parseExplicitRoleGroups(raw string) map[string]string {
	m := make(map[string]string)
	for entry := range strings.SplitSeq(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		idx := strings.IndexByte(entry, '=')
		if idx < 0 {
			slog.Warn("invalid ROLE_GROUPS entry, missing '='", "entry", entry)
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

func maskDSN(dsn string) string {
	if i := strings.Index(dsn, "@"); i > 0 {
		if j := strings.LastIndex(dsn[:i], ":"); j > 0 {
			return dsn[:j+1] + "***" + dsn[i:]
		}
	}
	return dsn
}

func mustGetEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.Error("required env missing", "key", key)
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

