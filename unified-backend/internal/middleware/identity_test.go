package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"unified-backend/internal/middleware"
)

var roleGroups = map[string]string{
	"editors":      "editor",
	"admins":       "admin",
	"moderators":   "moderator",
}

// ── parseGroups (через ExtractIdentity behavior) ──────────────────────────────

func TestExtractIdentity_NoSub_Returns401(t *testing.T) {
	mw := middleware.ExtractIdentity(roleGroups)
	handler := mw(okHandler())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("want 401, got %d", rec.Code)
	}
}

func TestExtractIdentity_WithSub_PassesThrough(t *testing.T) {
	mw := middleware.ExtractIdentity(roleGroups)
	handler := mw(okHandler())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Auth-Request-User", "user-sub-123")
	req.Header.Set("X-Auth-Request-Groups", "admins")
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestExtractIdentity_ResolvesRole(t *testing.T) {
	mw := middleware.ExtractIdentity(roleGroups)

	var capturedIdentity *middleware.Identity
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedIdentity = middleware.IdentityFromCtx(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Auth-Request-User", "sub-abc")
	req.Header.Set("X-Auth-Request-Email", "user@example.com")
	req.Header.Set("X-Auth-Request-Preferred-Username", "john")
	req.Header.Set("X-Auth-Request-Groups", "editors, admins")
	handler.ServeHTTP(rec, req)

	if capturedIdentity == nil {
		t.Fatal("identity not in context")
	}
	if capturedIdentity.Sub != "sub-abc" {
		t.Errorf("sub: want sub-abc, got %s", capturedIdentity.Sub)
	}
	if capturedIdentity.Email != "user@example.com" {
		t.Errorf("email: want user@example.com, got %s", capturedIdentity.Email)
	}
	// первое совпадение = editor (editors → editor)
	if capturedIdentity.Role != "editor" {
		t.Errorf("role: want editor, got %s", capturedIdentity.Role)
	}
	// все роли: editor + admin
	if len(capturedIdentity.Roles) != 2 {
		t.Errorf("roles count: want 2, got %d: %v", len(capturedIdentity.Roles), capturedIdentity.Roles)
	}
}

func TestExtractIdentity_NoGroupMatch_EmptyRole(t *testing.T) {
	mw := middleware.ExtractIdentity(roleGroups)
	var captured *middleware.Identity
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = middleware.IdentityFromCtx(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Auth-Request-User", "sub-xyz")
	req.Header.Set("X-Auth-Request-Groups", "unknown-group")
	handler.ServeHTTP(rec, req)

	if captured.Role != "" {
		t.Errorf("role should be empty for unmapped group, got %s", captured.Role)
	}
	if len(captured.Roles) != 0 {
		t.Errorf("roles should be empty, got %v", captured.Roles)
	}
}

func TestExtractIdentity_DeduplicatesRoles(t *testing.T) {
	mw := middleware.ExtractIdentity(roleGroups)
	var captured *middleware.Identity
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = middleware.IdentityFromCtx(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Auth-Request-User", "sub-dup")
	// два разных keycloak-группы, оба маппятся на editor
	req.Header.Set("X-Auth-Request-Groups", "editors, editors")
	handler.ServeHTTP(rec, req)

	if len(captured.Roles) != 1 {
		t.Errorf("duplicate roles should be deduplicated, got %v", captured.Roles)
	}
}

func TestExtractIdentity_EmptyGroups(t *testing.T) {
	mw := middleware.ExtractIdentity(roleGroups)
	var captured *middleware.Identity
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = middleware.IdentityFromCtx(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Auth-Request-User", "sub-nogroup")
	handler.ServeHTTP(rec, req)

	if len(captured.Groups) != 0 {
		t.Errorf("groups should be nil for empty header, got %v", captured.Groups)
	}
}

// ── ClientIP ──────────────────────────────────────────────────────────────────

func TestClientIP_XRealIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "1.2.3.4")
	if ip := middleware.ClientIP(req); ip != "1.2.3.4" {
		t.Errorf("want 1.2.3.4, got %s", ip)
	}
}

func TestClientIP_XForwardedFor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "5.6.7.8, 9.10.11.12")
	if ip := middleware.ClientIP(req); ip != "5.6.7.8" {
		t.Errorf("want 5.6.7.8, got %s", ip)
	}
}

func TestClientIP_RemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	if ip := middleware.ClientIP(req); ip != "10.0.0.1:1234" {
		t.Errorf("want 10.0.0.1:1234, got %s", ip)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}
