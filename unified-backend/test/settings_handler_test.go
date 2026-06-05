package test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"unified-backend/internal/config"
	"unified-backend/internal/handler"
	"unified-backend/internal/service"
	"unified-backend/internal/shlink"
)

// ── helpers ────────────────────────────────────────────────────────────────

func newSettingsHandler(cfg *config.Config) *handler.SettingsHandler {
	client := shlink.NewClient("http://localhost:9999") // unreachable — GetHealth will fail gracefully
	cache := service.NewPermissionsCache(nil, "")
	svc := service.NewShlinkService(client, cfg, cache)
	return handler.NewSettingsHandler(cfg, svc, nil)
}

func defaultCfg() *config.Config {
	return &config.Config{
		ShlinkShortIDLength:    6,
		UserCustomSlugEnabled:  true,
		UserSlugPrefixEnabled:  false,
		ShlinkDefaultDomain:    "https://s.example.com",
		ShlinkURL:              "http://shlink:8080",
	}
}

// ── GET /api/admin/settings ────────────────────────────────────────────────

func TestGetSettings_ReturnsCurrentConfigValues(t *testing.T) {
	cfg := defaultCfg()
	h := newSettingsHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/settings", nil)
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
	// connected=false because shlink is not reachable in tests — that's expected
}

func TestGetSettings_DomainFallbackToShlinkURL(t *testing.T) {
	cfg := defaultCfg()
	cfg.ShlinkDefaultDomain = "" // not set → fallback to ShlinkURL
	h := newSettingsHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/settings", nil)
	rec := httptest.NewRecorder()
	h.GetSettings(rec, req)

	var resp struct{ Domain string `json:"domain"` }
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Domain != cfg.ShlinkURL {
		t.Errorf("domain fallback: want %q, got %q", cfg.ShlinkURL, resp.Domain)
	}
}

// ── PATCH /api/admin/settings ──────────────────────────────────────────────

func TestPatchSettings_UpdatesAllFields(t *testing.T) {
	cfg := defaultCfg()
	h := newSettingsHandler(cfg)

	body := `{"shortCodeLength":10,"allowCustomSlugs":false,"userSlugPrefix":true,"domain":"https://new.example.com"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/admin/settings", strings.NewReader(body))
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
			h := newSettingsHandler(cfg)
			body := strings.NewReader(`{"shortCodeLength":` + itoa(tc.input) + `}`)
			req := httptest.NewRequest(http.MethodPatch, "/api/admin/settings", body)
			rec := httptest.NewRecorder()
			h.PatchSettings(rec, req)
			if cfg.ShlinkShortIDLength != tc.want {
				t.Errorf("input %d: want %d, got %d", tc.input, tc.want, cfg.ShlinkShortIDLength)
			}
		})
	}
}

func TestPatchSettings_PartialUpdate(t *testing.T) {
	cfg := defaultCfg()
	h := newSettingsHandler(cfg)

	// Only update one field — others must stay unchanged
	body := `{"allowCustomSlugs":false}`
	req := httptest.NewRequest(http.MethodPatch, "/api/admin/settings", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.PatchSettings(rec, req)

	if cfg.ShlinkShortIDLength != 6 {
		t.Errorf("shortCodeLength must not change, got %d", cfg.ShlinkShortIDLength)
	}
	if cfg.UserCustomSlugEnabled {
		t.Error("allowCustomSlugs: want false after patch")
	}
	if cfg.ShlinkDefaultDomain != "https://s.example.com" {
		t.Errorf("domain must not change, got %q", cfg.ShlinkDefaultDomain)
	}
}

func TestPatchSettings_InvalidJSON(t *testing.T) {
	cfg := defaultCfg()
	h := newSettingsHandler(cfg)

	req := httptest.NewRequest(http.MethodPatch, "/api/admin/settings", strings.NewReader("{not json"))
	rec := httptest.NewRecorder()
	h.PatchSettings(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestPatchSettings_EmptyDomainIgnored(t *testing.T) {
	cfg := defaultCfg()
	h := newSettingsHandler(cfg)

	body := `{"domain":""}`
	req := httptest.NewRequest(http.MethodPatch, "/api/admin/settings", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.PatchSettings(rec, req)

	// Empty string must not overwrite existing value
	if cfg.ShlinkDefaultDomain != "https://s.example.com" {
		t.Errorf("domain must not change on empty string, got %q", cfg.ShlinkDefaultDomain)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
