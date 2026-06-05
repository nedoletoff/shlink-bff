package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"unified-backend/internal/config"
	"unified-backend/internal/domain"
	"unified-backend/internal/handler"
	"unified-backend/internal/service"
)

// ── stubs ─────────────────────────────────────────────────────────────────────

type stubRolesRepo struct {
	data      []domain.RolePermissions
	upsertErr error
}

func (s *stubRolesRepo) GetAll(_ context.Context) ([]domain.RolePermissions, error) {
	return s.data, nil
}
func (s *stubRolesRepo) Upsert(_ context.Context, p *domain.RolePermissions) error {
	if s.upsertErr != nil {
		return s.upsertErr
	}
	for i, r := range s.data {
		if r.Role == p.Role {
			s.data[i] = *p
			return nil
		}
	}
	s.data = append(s.data, *p)
	return nil
}

func newTestCache(perms ...domain.RolePermissions) *service.PermissionsCache {
	repo := &stubRolesRepo{data: perms}
	c := service.NewPermissionsCache(repo, "admin")
	_ = c.Load(context.Background())
	return c
}

func testCfg() *config.Config {
	return &config.Config{
		AdminRole: "admin",
		RoleGroups: map[string]string{
			"editors": "editor",
			"admins":  "admin",
		},
	}
}

// ── ListRoles ─────────────────────────────────────────────────────────────────

func TestListRoles_ReturnsRolesAndMappings(t *testing.T) {
	cache := newTestCache(
		domain.RolePermissions{Role: "editor", CanCreateLinks: true},
	)
	repo := &stubRolesRepo{}
	h := handler.NewRolesHandler(cache, repo, testCfg())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/roles", nil)
	h.ListRoles(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if _, ok := resp["roles"]; !ok {
		t.Error("response should have roles key")
	}
	if _, ok := resp["mappings"]; !ok {
		t.Error("response should have mappings key")
	}
}

// ── GetRole ───────────────────────────────────────────────────────────────────

func TestGetRole_ExistingRole(t *testing.T) {
	cache := newTestCache(
		domain.RolePermissions{Role: "editor", CanCreateLinks: true, CanEditOwnLinks: true},
	)
	h := handler.NewRolesHandler(cache, &stubRolesRepo{}, testCfg())

	rec := httptest.NewRecorder()
	req := chiRequest(http.MethodGet, "/api/admin/roles/editor", nil, chi.RouteParams{
		Keys: []string{"role"}, Values: []string{"editor"},
	})
	h.GetRole(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	var p domain.RolePermissions
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p.Role != "editor" {
		t.Errorf("want editor, got %s", p.Role)
	}
	if !p.CanCreateLinks {
		t.Error("should have CanCreateLinks")
	}
}

func TestGetRole_UnknownRole_ReturnsFallback(t *testing.T) {
	cache := newTestCache()
	h := handler.NewRolesHandler(cache, &stubRolesRepo{}, testCfg())

	rec := httptest.NewRecorder()
	req := chiRequest(http.MethodGet, "/api/admin/roles/ghost", nil, chi.RouteParams{
		Keys: []string{"role"}, Values: []string{"ghost"},
	})
	h.GetRole(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	var p domain.RolePermissions
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p.Role != "ghost" {
		t.Errorf("role field should be ghost, got %s", p.Role)
	}
}

// ── UpsertRolePermissions ─────────────────────────────────────────────────────

func TestUpsertRolePermissions_Valid(t *testing.T) {
	cache := newTestCache()
	repo := &stubRolesRepo{}
	h := handler.NewRolesHandler(cache, repo, testCfg())

	body, _ := json.Marshal(map[string]bool{
		"canCreateLinks":  true,
		"canViewOwnLinks": true,
	})
	rec := httptest.NewRecorder()
	req := chiRequest(http.MethodPut, "/api/admin/roles/editor/permissions", bytes.NewReader(body), chi.RouteParams{
		Keys: []string{"role"}, Values: []string{"editor"},
	})
	h.UpsertRolePermissions(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
	p := cache.Get("editor")
	if !p.CanCreateLinks || !p.CanViewOwnLinks {
		t.Errorf("cache not updated: %+v", p)
	}
}

func TestUpsertRolePermissions_InvalidJSON(t *testing.T) {
	cache := newTestCache()
	h := handler.NewRolesHandler(cache, &stubRolesRepo{}, testCfg())

	rec := httptest.NewRecorder()
	req := chiRequest(http.MethodPut, "/api/admin/roles/editor/permissions", bytes.NewReader([]byte(`not-json`)), chi.RouteParams{
		Keys: []string{"role"}, Values: []string{"editor"},
	})
	h.UpsertRolePermissions(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", rec.Code)
	}
}

func TestUpsertRolePermissions_RepoError(t *testing.T) {
	cache := newTestCache()
	errRepo := &stubRolesRepo{upsertErr: errHandlerStub}
	h := handler.NewRolesHandler(cache, errRepo, testCfg())

	body, _ := json.Marshal(map[string]bool{"canCreateLinks": true})
	rec := httptest.NewRecorder()
	req := chiRequest(http.MethodPut, "/api/admin/roles/editor/permissions", bytes.NewReader(body), chi.RouteParams{
		Keys: []string{"role"}, Values: []string{"editor"},
	})
	h.UpsertRolePermissions(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", rec.Code)
	}
}

// ── permToStringSlice (через ListRoles) ───────────────────────────────────────

func TestPermToStringSlice_OnlyTrueFlags(t *testing.T) {
	cache := newTestCache(
		domain.RolePermissions{
			Role:            "tester",
			CanCreateLinks:  true,
			CanViewOwnLinks: true,
		},
	)
	h := handler.NewRolesHandler(cache, &stubRolesRepo{}, testCfg())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/roles", nil)
	h.ListRoles(rec, req)

	var resp struct {
		Roles []struct {
			Role        string   `json:"role"`
			Permissions []string `json:"permissions"`
		} `json:"roles"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	for _, r := range resp.Roles {
		if r.Role == "tester" {
			if len(r.Permissions) != 2 {
				t.Errorf("want 2 permissions, got %d: %v", len(r.Permissions), r.Permissions)
			}
			for _, p := range r.Permissions {
				if p != "canCreateLinks" && p != "canViewOwnLinks" {
					t.Errorf("unexpected permission: %s", p)
				}
			}
		}
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

var errHandlerStub = &handlerErr{"stub"}

type handlerErr struct{ msg string }

func (e *handlerErr) Error() string { return e.msg }

func chiRequest(method, url string, body io.Reader, params chi.RouteParams) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, url, body)
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	rctx := chi.NewRouteContext()
	rctx.URLParams = params
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}
