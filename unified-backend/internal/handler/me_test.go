package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"unified-backend/internal/config"
	"unified-backend/internal/domain"
	"unified-backend/internal/handler"
	"unified-backend/internal/middleware"
)

// stubPermSvc implements a minimal PermissionService-like interface for tests.
type stubPermSvc struct {
	perms []string
	err   error
}

func (s *stubPermSvc) GetUserPermissions(_ context.Context, _ string) ([]string, error) {
	return s.perms, s.err
}

func (s *stubPermSvc) UserHasPermission(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

func meCfg(slugPrefix bool) *config.Config {
	return &config.Config{
		AdminRole:                "admin",
		UserSlugPrefixEnabled:    slugPrefix,
		UserTagInternalIdEnabled: true,
	}
}

func TestMeHandler_NoUser_Returns500(t *testing.T) {
	svc := &stubPermSvc{perms: []string{}}
	h := handler.NewMeHandler(meCfg(false), svc)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", rec.Code)
	}
}

func TestMeHandler_WithUser_Returns200(t *testing.T) {
	svc := &stubPermSvc{perms: []string{"short_urls.create", "dashboard.view"}}
	h := handler.NewMeHandler(meCfg(false), svc)

	u := &domain.User{
		ID:           "uuid-1",
		Sub:          "sub-1",
		Username:     "john",
		Email:        "john@example.com",
		Role:         "editor",
		ShlinkAPIKey: "key-123",
	}
	ctx := middleware.WithUser(context.Background(), u)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil).WithContext(ctx)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["sub"] != "sub-1" {
		t.Errorf("sub: want sub-1, got %v", resp["sub"])
	}
	if resp["hasApiKey"] != true {
		t.Errorf("hasApiKey: want true, got %v", resp["hasApiKey"])
	}
	// Проверяем что isAdmin отсутствует
	if _, ok := resp["isAdmin"]; ok {
		t.Error("isAdmin must not be present in /api/me response")
	}
}

func TestMeHandler_PermissionsArray_Returned(t *testing.T) {
	expected := []string{"short_urls.create", "dashboard.view", "users.view"}
	svc := &stubPermSvc{perms: expected}
	h := handler.NewMeHandler(meCfg(false), svc)

	u := &domain.User{ID: "uuid-2", Sub: "sub-2", Role: "editor"}
	ctx := middleware.WithUser(context.Background(), u)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil).WithContext(ctx)
	h.ServeHTTP(rec, req)

	var resp struct {
		Permissions []string `json:"permissions"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)

	if len(resp.Permissions) != len(expected) {
		t.Errorf("permissions: want %d items, got %d", len(expected), len(resp.Permissions))
	}
	permSet := make(map[string]bool, len(resp.Permissions))
	for _, p := range resp.Permissions {
		permSet[p] = true
	}
	for _, p := range expected {
		if !permSet[p] {
			t.Errorf("permissions: missing %q", p)
		}
	}
}

func TestMeHandler_SlugPrefix_IncludedWhenEnabled(t *testing.T) {
	svc := &stubPermSvc{perms: []string{}}
	h := handler.NewMeHandler(meCfg(true), svc)

	u := &domain.User{ID: "uuid-3", Sub: "sub-3", Role: "editor", SlugPrefix: "myprefix"}
	ctx := middleware.WithUser(context.Background(), u)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil).WithContext(ctx)
	h.ServeHTTP(rec, req)

	var resp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["slugPrefix"] != "myprefix" {
		t.Errorf("slugPrefix: want myprefix, got %v", resp["slugPrefix"])
	}
}

func TestMeHandler_SlugPrefix_OmittedWhenDisabled(t *testing.T) {
	svc := &stubPermSvc{perms: []string{}}
	h := handler.NewMeHandler(meCfg(false), svc)

	u := &domain.User{ID: "uuid-4", Sub: "sub-4", Role: "editor", SlugPrefix: "myprefix"}
	ctx := middleware.WithUser(context.Background(), u)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil).WithContext(ctx)
	h.ServeHTTP(rec, req)

	var resp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if _, ok := resp["slugPrefix"]; ok {
		t.Errorf("slugPrefix should be omitted when disabled, got %v", resp["slugPrefix"])
	}
}

func TestMeHandler_NoAPIKey_HasApikeyFalse(t *testing.T) {
	svc := &stubPermSvc{perms: []string{}}
	h := handler.NewMeHandler(meCfg(false), svc)

	u := &domain.User{ID: "uuid-5", Sub: "sub-5", Role: "editor", ShlinkAPIKey: ""}
	ctx := middleware.WithUser(context.Background(), u)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil).WithContext(ctx)
	h.ServeHTTP(rec, req)

	var resp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["hasApiKey"] != false {
		t.Errorf("hasApiKey: want false, got %v", resp["hasApiKey"])
	}
}
