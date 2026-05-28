package config

import (
	"log/slog"
	"os"
	"strconv"
)

type Config struct {
	HTTPAddr    string
	DatabaseURL string
	ShlinkURL   string

	// Feature flags
	UserSlugPrefixEnabled    bool
	UserTagInternalIdEnabled bool
}

func Load() *Config {
	cfg := &Config{
		HTTPAddr:                 getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:              mustGetEnv("DATABASE_URL"),
		ShlinkURL:                mustGetEnv("SHLINK_INTERNAL_URL"),
		UserSlugPrefixEnabled:    getBool("FEATURE_USER_SLUG_PREFIX", false),
		UserTagInternalIdEnabled: getBool("FEATURE_USER_TAG_INTERNAL_ID", false),
	}
	return cfg
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
