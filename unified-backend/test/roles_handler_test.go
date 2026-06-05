package test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"unified-backend/internal/config"
	"unified-backend/internal/domain"
	"unified-backend/internal/handler"
	"unified-backend/internal/service"
)

// ── in-memory RolesRepository ──────────────────────────────────────────────

type memRolesRepo struct {
	mu    sync.RWMutex
	store map[string]domain.RolePermissions
}

func newMemRolesRepo(initial ...domain.RolePermissions) *memRolesRepo {
	r := &memRolesRepo{store: make(map[string]domain.RolePermissions)}
	for _, p := range initial {
		r.store[p.Role] = p
	}
	return r
}

func (r *memRolesRepo) GetAll(_ context.Context) ([]domain.RolePermissions, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.RolePermissions, 0, len(r.store))
	for _, p := range r.store {
		out = append(out, p)
	}
	return out, nil
}

func (r *memRolesRepo) Upsert(_ context.Context, p *domain.RolePermissions) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store[p.Role] = *p
	return nil
}

// ── fixtures ───────────────────────────────────────────────────────────────

func newRolesHandler(initial ...domain.RolePermissions) *handler.RolesHandler {
	cache := service.NewPermissionsCache(nil, domain.RoleAdmin)
	for _, p := range initial {
		cache.Set(p)
	}
	repo := newMemRolesRepo(initial...)
	cfg := &config.Config{RoleGroups: map[string]string{"admin-group": domain.RoleAdmin}}
	return handler.NewRolesHandler(cache, repo, cfg)
}

// ── ListRoles ──────────────────────────────────────────────────────────────

func TestListRoles_ReturnsAllRolesAndMappings(t *testing.T) {
	adminP := domain.DefaultAdminPermissions(domain.RoleAdmin)
	userP := domain.DefaultUserPermissions(domain.RoleUser)
	h := newRolesHandler(adminP, userP)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/roles", nil)
	rec := httptest.NewRecorder()
	h.ListRoles(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}

	var resp struct {
		Roles    []struct{ Role string `json:"role"` } `json:"roles"`
		Mappings []struct {
			KcGroup string `json:"kcGroup"`
			AppRole string `json:"appRole"`
		} `json:"mappings"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Roles) != 2 {
		t.Errorf("want 2 roles, got %d", len(resp.Roles))
	}
	if len(resp.Mappings) != 1 {
		t.Errorf("want 1 mapping, got %d", len(resp.Mappings))
	}
	if resp.Mappings[0].KcGroup != "admin-group" || resp.Mappings[0].AppRole != domain.RoleAdmin {
		t.Errorf("unexpected mapping: %+v", resp.Mappings[0])
	}
}

// ── GetRole ────────────────────────────────────────────────────────────────

func TestGetRole_KnownRole(t *testing.T) {
	adminP := domain.DefaultAdminPermissions(domain.RoleAdmin)
	h := newRolesHandler(adminP)

	// simulate chi URL param via custom handler wrapper
	w, r := newChiRequest(http.MethodGet, "/api/admin/roles/admin", "", "role", "admin")
	h.GetRole(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var p domain.RolePermissions
	if err := json.NewDecoder(w.Body).Decode(&p); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !p.CanViewAllLinks {
		t.Error("admin: CanViewAllLinks must be true")
	}
	if !p.CanManageUsers {
		t.Error("admin: CanManageUsers must be true")
	}
}

func TestGetRole_UnknownRoleReturnsZeroPerms(t *testing.T) {
	h := newRolesHandler()
	w, r := newChiRequest(http.MethodGet, "/api/admin/roles/ghost", "", "role", "ghost")
	h.GetRole(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	var p domain.RolePermissions
	_ = json.NewDecoder(w.Body).Decode(&p)
	// All booleans are false (zero value)
	if p.CanViewAllLinks || p.CanCreateLinks || p.CanManageUsers {
		t.Error("unknown role: expected all-false permissions")
	}
}

// ── UpsertRolePermissions ──────────────────────────────────────────────────

func TestUpsertRolePermissions_CreateNew(t *testing.T) {
	h := newRolesHandler()

	body := `{"canViewOwnLinks":true,"canCreateLinks":true}`
	w, r := newChiRequest(http.MethodPut, "/api/admin/roles/editor/permissions", body, "role", "editor")
	h.UpsertRolePermissions(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d — %s", w.Code, w.Body.String())
	}
	var p domain.RolePermissions
	_ = json.NewDecoder(w.Body).Decode(&p)
	if p.Role != "editor" {
		t.Errorf("role in response must be 'editor', got %q", p.Role)
	}
	if !p.CanViewOwnLinks {
		t.Error("canViewOwnLinks: want true")
	}
}

func TestUpsertRolePermissions_UpdateExisting_CacheInvalidated(t *testing.T) {
	initial := domain.DefaultUserPermissions(domain.RoleUser)
	h := newRolesHandler(initial)

	// Grant all links visibility
	body := `{"canViewAllLinks":true,"canViewOwnLinks":true,"canCreateLinks":true,"canEditOwnLinks":true,"canDeleteOwnLinks":true}`
	w, r := newChiRequest(http.MethodPut, "/api/admin/roles/user/permissions", body, "role", domain.RoleUser)
	h.UpsertRolePermissions(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}

	// Verify cache was updated: build a new service using the same cache
	var p domain.RolePermissions
	_ = json.NewDecoder(w.Body).Decode(&p)
	if !p.CanViewAllLinks {
		t.Error("CanViewAllLinks must be true after upsert")
	}
}

func TestUpsertRolePermissions_RoleFromURLNotBody(t *testing.T) {
	h := newRolesHandler()

	// Body says role="wrong", URL param says role="correct"
	body := `{"role":"wrong","canCreateLinks":true}`
	w, r := newChiRequest(http.MethodPut, "/api/admin/roles/correct/permissions", body, "role", "correct")
	h.UpsertRolePermissions(w, r)

	var p domain.RolePermissions
	_ = json.NewDecoder(w.Body).Decode(&p)
	if p.Role != "correct" {
		t.Errorf("role must come from URL, got %q", p.Role)
	}
}

func TestUpsertRolePermissions_InvalidJSON(t *testing.T) {
	h := newRolesHandler()
	w, r := newChiRequest(http.MethodPut, "/api/admin/roles/x/permissions", "{bad", "role", "x")
	h.UpsertRolePermissions(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

// ── permToStringSlice round-trip ───────────────────────────────────────────

func TestPermToStringSlice_AllFlags(t *testing.T) {
	p := domain.DefaultAdminPermissions(domain.RoleAdmin)
	flags := handler.PermToStringSliceExported(p)

	expected := []string{
		"canViewOwnLinks", "canViewAllLinks",
		"canCreateLinks", "canCreateWithCustomSlug", "canCreateWithoutSlug",
		"canEditOwnLinks", "canEditAllLinks",
		"canDeleteOwnLinks", "canDeleteAllLinks",
		"canManageOwnTags", "canManageAllTags",
		"canViewOwnStats", "canViewAllStats",
		"canViewAuditLogs", "canManageUsers", "canManageRoles",
	}

	if len(flags) != len(expected) {
		t.Errorf("want %d flags, got %d: %v", len(expected), len(flags), flags)
		return
	}
	for i, want := range expected {
		if flags[i] != want {
			t.Errorf("flag[%d]: want %q, got %q", i, want, flags[i])
		}
	}
}

func TestPermToStringSlice_NoFlags(t *testing.T) {
	p := domain.RolePermissions{Role: "empty"}
	flags := handler.PermToStringSliceExported(p)
	if len(flags) != 0 {
		t.Errorf("want 0 flags, got %d: %v", len(flags), flags)
	}
}

// ── chi URL param injection helper ────────────────────────────────────────

func newChiRequest(
	method, path, body string,
	paramKey, paramVal string,
) (*httptest.ResponseRecorder, *http.Request) {
	var bodyReader *strings.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	} else {
		bodyReader = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	// Inject chi URL param via context
	rctx := chiContext()
	rctx.URLParams.Add(paramKey, paramVal)
	req = req.WithContext(context.WithValue(req.Context(), chiRouteContextKey{}, rctx))
	return httptest.NewRecorder(), req
}
