package config

import (
	"os"
	"testing"
)

func TestLoad_ShlinkRunnerDefaults(t *testing.T) {
	withEnv(t, map[string]string{
		"SHLINK_INTERNAL_URL": "http://shlink-api:8080",
		"DB_USER":             "bff",
		"DB_PASSWORD":         "secret",
		"DB_NAME":             "shlink_bff",
	}, func() {
		cfg := Load()
		if cfg.ShlinkRunnerMode != "docker" {
			t.Fatalf("expected ShlinkRunnerMode=docker, got %q", cfg.ShlinkRunnerMode)
		}
		if cfg.ShlinkContainerName != "shlink-api" {
			t.Fatalf("expected ShlinkContainerName=shlink-api, got %q", cfg.ShlinkContainerName)
		}
		if cfg.ShlinkBin != "shlink" {
			t.Fatalf("expected ShlinkBin=shlink, got %q", cfg.ShlinkBin)
		}
	})
}

func TestLoad_ShlinkRunnerOverrides(t *testing.T) {
	withEnv(t, map[string]string{
		"SHLINK_INTERNAL_URL":      "http://shlink-api:8080",
		"DB_USER":                  "bff",
		"DB_PASSWORD":              "secret",
		"DB_NAME":                  "shlink_bff",
		"SHLINK_RUNNER_MODE":       "native",
		"SHLINK_CONTAINER":         "custom-shlink",
		"SHLINK_BIN":               "/usr/local/bin/shlink",
		"ROLE_SOURCE":              "db",
		"FEATURE_USER_CUSTOM_SLUG": "false",
		"SHLINK_SHORT_ID_LENGTH":   "12",
	}, func() {
		cfg := Load()
		if cfg.ShlinkRunnerMode != "native" {
			t.Fatalf("expected ShlinkRunnerMode=native, got %q", cfg.ShlinkRunnerMode)
		}
		if cfg.ShlinkContainerName != "custom-shlink" {
			t.Fatalf("expected ShlinkContainerName=custom-shlink, got %q", cfg.ShlinkContainerName)
		}
		if cfg.ShlinkBin != "/usr/local/bin/shlink" {
			t.Fatalf("expected ShlinkBin=/usr/local/bin/shlink, got %q", cfg.ShlinkBin)
		}
		if cfg.RoleSource != RoleSourceDB {
			t.Fatalf("expected RoleSource=db, got %q", cfg.RoleSource)
		}
		if cfg.UserCustomSlugEnabled {
			t.Fatalf("expected UserCustomSlugEnabled=false")
		}
		if cfg.ShlinkShortIDLength != 12 {
			t.Fatalf("expected ShlinkShortIDLength=12, got %d", cfg.ShlinkShortIDLength)
		}
	})
}

func withEnv(t *testing.T, vars map[string]string, fn func()) {
	t.Helper()
	old := make(map[string]*string, len(vars))
	for k, v := range vars {
		if cur, ok := os.LookupEnv(k); ok {
			c := cur
			old[k] = &c
		} else {
			old[k] = nil
		}
		if err := os.Setenv(k, v); err != nil {
			t.Fatalf("os.Setenv(%s): %v", k, err)
		}
	}
	defer func() {
		for k, v := range old {
			if v == nil {
				_ = os.Unsetenv(k)
			} else {
				_ = os.Setenv(k, *v)
			}
		}
	}()
	fn()
}
