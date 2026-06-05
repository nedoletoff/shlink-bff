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
	"unified-backend/internal/service"
)

func newMeCache(perms ...domain.RolePermissions) *service.PermissionsCache {
	repo := &stubRolesRepo{data: perms}
	c := service.NewPermissionsCache(repo, "admin")
	_ = c.Load(context.Background())
	return c
}

func meCfg(slugPrefix bool) *config.Config {
	return &config.Config{
		AdminRole:                "admin",
		UserSlugPrefixEnabled:    slugPrefix,
		UserTagInternalIdEnabled: true,
	}
}

func TestMeHandler_NoUser_Returns500(t *testing.T) {
	h := handler.NewMeHandler(meCfg(false), newMeCache())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", rec.Code)
	}
}

func TestMeHandler_WithUser_Returns200(t *testing.T) {
	cache := newMeCache(domain.RolePermissions{
		Role:           "editor",
		CanCreateLinks: true,
		CanEditOwnLinks: true,
	})
	h := handler.NewMeHandler(meCfg(false), cache)

	u := &domain.User{
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
	if resp["username"] != "john" {
		t.Errorf("username: want john, got %v", resp["username"])
	}
	if resp["hasApiKey"] != true {
		t.Errorf("hasApiKey: want true, got %v", resp["hasApiKey"])
	}
}

func TestMeHandler_SlugPrefix_IncludedWhenEnabled(t *testing.T) {
	cache := newMeCache(domain.RolePermissions{Role: "editor"})
	h := handler.NewMeHandler(meCfg(true), cache)

	u := &domain.User{
		Sub:        "sub-2",
		Role:       "editor",
		SlugPrefix: "myprefix",
	}
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
	cache := newMeCache(domain.RolePermissions{Role: "editor"})
	h := handler.NewMeHandler(meCfg(false), cache)

	u := &domain.User{
		Sub:        "sub-3",
		Role:       "editor",
		SlugPrefix: "myprefix",
	}
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
	cache := newMeCache(domain.RolePermissions{Role: "editor"})
	h := handler.NewMeHandler(meCfg(false), cache)

	u := &domain.User{Sub: "sub-4", Role: "editor", ShlinkAPIKey: ""}
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

func TestMeHandler_PermissionsMap_ContainsExpectedKeys(t *testing.T) {
	cache := newMeCache(domain.RolePermissions{
		Role:             "editor",
		CanCreateLinks:   true,
		CanManageUsers:   false,
		CanViewAuditLogs: false,
	})
	h := handler.NewMeHandler(meCfg(false), cache)

	u := &domain.User{Sub: "sub-5", Role: "editor"}
	ctx := middleware.WithUser(context.Background(), u)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil).WithContext(ctx)
	h.ServeHTTP(rec, req)

	var resp struct {
		Permissions map[string]bool `json:"permissions"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)

	expectedKeys := []string{
		"canCreateShortUrl", "canEditOwnLinks", "canDeleteOwnLinks",
		"canManageOwnTags", "canViewAuditLogs", "canManageUsers",
	}
	for _, k := range expectedKeys {
		if _, ok := resp.Permissions[k]; !ok {
			t.Errorf("permissions map missing key: %s", k)
		}
	}
	if !resp.Permissions["canCreateShortUrl"] {
		t.Error("canCreateShortUrl should be true for editor")
	}
	if resp.Permissions["canManageUsers"] {
		t.Error("canManageUsers should be false for editor")
	}
}
