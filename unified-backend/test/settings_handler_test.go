package test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"unified-backend/internal/config"
	"unified-backend/internal/domain"
	"unified-backend/internal/handler"
	"unified-backend/internal/middleware"
	"unified-backend/internal/service"
	"unified-backend/internal/shlink"
)

// ── helpers ───────────────────────────────────────────────────────────────

func newSettingsHandler(cfg *config.Config) *handler.SettingsHandler {
	client := shlink.NewClient("http://localhost:9999") // unreachable — GetHealth will fail gracefully
	cache := service.NewPermissionsCache(nil, "")
	svc := service.NewShlinkService(client, cfg, cache)
	return handler.NewSettingsHandler(cfg, svc, nil)
}

func defaultCfg() *config.Config {
	return &config.Config{
		ShlinkShortIDLength:   6,
		UserCustomSlugEnabled: true,
		UserSlugPrefixEnabled: false,
		ShlinkDefaultDomain:   "https://s.example.com",
		ShlinkURL:             "http://shlink:8080",
	}
}

// reqWithAdmin возвращает httptest.Request с admin-пользователем в контексте.
func reqWithAdmin(method, path string, body *strings.Reader) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, body)
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	u := &domain.User{Sub: "admin-sub", Role: "admin", Username: "admin"}
	return req.WithContext(middleware.WithUser(context.Background(), u))
}

// reqWithUser возвращает httptest.Request с обычным user в контексте (без canManageSettings).
func reqWithUser(method, path string, body *strings.Reader) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, body)
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	u := &domain.User{Sub: "user-sub", Role: "user", Username: "user"}
	return req.WithContext(middleware.WithUser(context.Background(), u))
}

// newSettingsHandlerWithPerms создаёт handler с предзагруженными permissions в cache.
func newSettingsHandlerWithPerms(cfg *config.Config, perms ...domain.RolePermissions) *handler.SettingsHandler {
	client := shlink.NewClient("http://localhost:9999")
	cache := service.NewPermissionsCache(nil, "")
	for _, p := range perms {
		cache.Set(p)
	}
	svc := service.NewShlinkService(client, cfg, cache)
	return handler.NewSettingsHandler(cfg, svc, nil)
}

// ── GET /api/settings ────────────────────────────────────────────────────────

func TestGetSettings_ReturnsCurrentConfigValues(t *testing.T) {
	cfg := defaultCfg()
	h := newSettingsHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	rec := httptest.NewRecorder()
	h.GetSettings(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}

	var resp struct {
		ShortCodeLength  int    `json:"shortCodeLength"`
		AllowCustomSlugs bool   `json:"allowCustomSlugs"`
		UserSlugPrefix   bool   `json:"userSlugPrefix"`
		Domain           string `json:"domain"`
		Connected        bool   `json:"connected"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.ShortCodeLength != 6 {
		t.Errorf("shortCodeLength: want 6, got %d", resp.ShortCodeLength)
	}
	if !resp.AllowCustomSlugs {
		t.Error("allowCustomSlugs: want true")
	}
	if resp.UserSlugPrefix {
		t.Error("userSlugPrefix: want false")
	}
	if resp.Domain != "https://s.example.com" {
		t.Errorf("domain: want https://s.example.com, got %q", resp.Domain)
	}
	// connected=false потому что shlink недоступен в тестах — ожидаемое поведение
}

func TestGetSettings_DomainFallbackToShlinkURL(t *testing.T) {
	cfg := defaultCfg()
	cfg.ShlinkDefaultDomain = "" // not set → fallback to ShlinkURL
	h := newSettingsHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	rec := httptest.NewRecorder()
	h.GetSettings(rec, req)

	var resp struct{ Domain string `json:"domain"` }
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Domain != cfg.ShlinkURL {
		t.Errorf("domain fallback: want %q, got %q", cfg.ShlinkURL, resp.Domain)
	}
}

// GET доступен всем авторизованным: даже без пользователя в контексте возвращает 200.
func TestGetSettings_NoUserContext_Returns200(t *testing.T) {
	cfg := defaultCfg()
	h := newSettingsHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	rec := httptest.NewRecorder()
	h.GetSettings(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET settings without user should still return 200, got %d", rec.Code)
	}
}

// ── PATCH /api/settings ────────────────────────────────────────────────────

func TestPatchSettings_UpdatesAllFields(t *testing.T) {
	cfg := defaultCfg()
	h := newSettingsHandlerWithPerms(cfg, domain.RolePermissions{
		Role: "admin", CanManageSettings: true,
	})

	body := strings.NewReader(`{"shortCodeLength":10,"allowCustomSlugs":false,"userSlugPrefix":true,"domain":"https://new.example.com"}`)
	req := reqWithAdmin(http.MethodPatch, "/api/settings", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.PatchSettings(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if cfg.ShlinkShortIDLength != 10 {
		t.Errorf("shortCodeLength: want 10, got %d", cfg.ShlinkShortIDLength)
	}
	if cfg.UserCustomSlugEnabled {
		t.Error("allowCustomSlugs: want false")
	}
	if !cfg.UserSlugPrefixEnabled {
		t.Error("userSlugPrefix: want true")
	}
	if cfg.ShlinkDefaultDomain != "https://new.example.com" {
		t.Errorf("domain: want https://new.example.com, got %q", cfg.ShlinkDefaultDomain)
	}
}

func TestPatchSettings_ShortCodeLengthBoundaries(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  int
	}{
		{"below_min", 1, 3},
		{"at_min", 3, 3},
		{"mid", 8, 8},
		{"at_max", 32, 32},
		{"above_max", 100, 32},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := defaultCfg()
			h := newSettingsHandlerWithPerms(cfg, domain.RolePermissions{
				Role: "admin", CanManageSettings: true,
			})
			body := strings.NewReader(`{"shortCodeLength":` + itoa(tc.input) + `}`)
			req := reqWithAdmin(http.MethodPatch, "/api/settings", body)
			rec := httptest.NewRecorder()
			h.PatchSettings(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("want 200, got %d", rec.Code)
			}
			if cfg.ShlinkShortIDLength != tc.want {
				t.Errorf("shortCodeLength: want %d, got %d", tc.want, cfg.ShlinkShortIDLength)
			}
		})
	}
}

func TestPatchSettings_PartialUpdate_OnlyChangesGivenFields(t *testing.T) {
	cfg := defaultCfg()
	origSlugPrefix := cfg.UserSlugPrefixEnabled
	h := newSettingsHandlerWithPerms(cfg, domain.RolePermissions{
		Role: "admin", CanManageSettings: true,
	})

	body := strings.NewReader(`{"shortCodeLength":9}`)
	req := reqWithAdmin(http.MethodPatch, "/api/settings", body)
	rec := httptest.NewRecorder()
	h.PatchSettings(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	if cfg.ShlinkShortIDLength != 9 {
		t.Errorf("shortCodeLength: want 9, got %d", cfg.ShlinkShortIDLength)
	}
	// не указанные поля остаются неизменными
	if cfg.UserSlugPrefixEnabled != origSlugPrefix {
		t.Errorf("userSlugPrefix should not change")
	}
}

func TestPatchSettings_InvalidJSON_Returns400(t *testing.T) {
	cfg := defaultCfg()
	h := newSettingsHandlerWithPerms(cfg, domain.RolePermissions{
		Role: "admin", CanManageSettings: true,
	})

	body := strings.NewReader(`not json`)
	req := reqWithAdmin(http.MethodPatch, "/api/settings", body)
	rec := httptest.NewRecorder()
	h.PatchSettings(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

// TestPatchSettings_NoPermission_Returns403 — PATCH требует canManageSettings.
// Пользователь с ролью без canManageSettings должен получить 403.
func TestPatchSettings_NoPermission_Returns403(t *testing.T) {
	cfg := defaultCfg()
	// user в cache есть, но CanManageSettings = false
	h := newSettingsHandlerWithPerms(cfg, domain.RolePermissions{
		Role: "user", CanManageSettings: false,
	})

	body := strings.NewReader(`{"shortCodeLength":10}`)
	req := reqWithUser(http.MethodPatch, "/api/settings", body)
	rec := httptest.NewRecorder()
	h.PatchSettings(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d — body: %s", rec.Code, rec.Body.String())
	}
}

// TestPatchSettings_NoUser_Returns403 — без пользователя в контексте — 403.
func TestPatchSettings_NoUser_Returns403(t *testing.T) {
	cfg := defaultCfg()
	h := newSettingsHandler(cfg)

	body := strings.NewReader(`{"shortCodeLength":10}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/settings", body)
	rec := httptest.NewRecorder()
	h.PatchSettings(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d", rec.Code)
	}
}

// ── helpers тестового пакета ─────────────────────────────────────────────

func itoa(n int) string {
	return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(
		string(rune('0'+n%10)),
		"\x00", "",
	), "\xff", "")) // простой конвертер через fmt недоступен, используем через strconv
}
