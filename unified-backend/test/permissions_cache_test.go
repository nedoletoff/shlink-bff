package test

import (
	"context"
	"testing"

	"unified-backend/internal/domain"
	"unified-backend/internal/service"
)

// newCache создаёт PermissionsCache с предзагруженными данными (без реальной БД).
func newCache(t *testing.T, rows []domain.RolePermissions, adminRole string) *service.PermissionsCache {
	t.Helper()
	cache := service.NewPermissionsCache(
		newStubRepo(rows),
		adminRole,
	)
	if err := cache.Load(context.Background()); err != nil {
		t.Fatalf("cache.Load: %v", err)
	}
	return cache
}

// --- Set + Get ---

func TestPermissionsCache_SetUpdatesCache(t *testing.T) {
	cache := newCache(t, nil, "admin")

	before := cache.Get("editor")
	if before.CanManageUsers {
		t.Fatal("editor should not have CanManageUsers before Set")
	}

	custom := domain.RolePermissions{
		Role:            "editor",
		CanViewOwnLinks: true,
		CanCreateLinks:  true,
		CanManageUsers:  true,
	}
	cache.Set(custom)

	after := cache.Get("editor")
	if after.Role != "editor" {
		t.Errorf("Role: got %q", after.Role)
	}
	if !after.CanCreateLinks {
		t.Error("CanCreateLinks should be true after Set")
	}
	if !after.CanManageUsers {
		t.Error("CanManageUsers should be true after Set")
	}
}

func TestPermissionsCache_SetOverwritesExisting(t *testing.T) {
	initial := domain.RolePermissions{
		Role:            "editor",
		CanCreateLinks:  true,
		CanEditOwnLinks: true,
	}
	cache := newCache(t, []domain.RolePermissions{initial}, "admin")

	updated := domain.RolePermissions{
		Role:            "editor",
		CanCreateLinks:  false,
		CanEditOwnLinks: true,
	}
	cache.Set(updated)

	got := cache.Get("editor")
	if got.CanCreateLinks {
		t.Error("CanCreateLinks should be false after overwrite")
	}
	if !got.CanEditOwnLinks {
		t.Error("CanEditOwnLinks should still be true")
	}
}

// --- Get fallback ---

func TestPermissionsCache_Get_AdminFallback(t *testing.T) {
	cache := newCache(t, nil, "admin")
	p := cache.Get("admin")

	if !p.CanManageUsers {
		t.Error("admin fallback: CanManageUsers should be true")
	}
	if !p.CanManageRoles {
		t.Error("admin fallback: CanManageRoles should be true")
	}
	if p.Role != "admin" {
		t.Errorf("admin fallback: Role should be admin, got %q", p.Role)
	}
}

func TestPermissionsCache_Get_UnknownRoleFallback(t *testing.T) {
	cache := newCache(t, nil, "admin")
	p := cache.Get("stranger")

	if p.CanManageUsers {
		t.Error("unknown role fallback: CanManageUsers should be false")
	}
	if p.CanManageRoles {
		t.Error("unknown role fallback: CanManageRoles should be false")
	}
	if !p.CanViewOwnLinks {
		t.Error("unknown role fallback: CanViewOwnLinks should be true")
	}
}

// --- GetMerged (OR-семантика) ---

func TestPermissionsCache_GetMerged_OR(t *testing.T) {
	roles := []domain.RolePermissions{
		{
			Role:            "reader",
			CanViewOwnLinks: true,
			CanViewOwnStats: true,
		},
		{
			Role:            "creator",
			CanCreateLinks:  true,
			CanEditOwnLinks: true,
		},
	}
	cache := newCache(t, roles, "admin")

	merged := cache.GetMerged([]string{"reader", "creator"})

	if !merged.CanViewOwnLinks {
		t.Error("merged: CanViewOwnLinks should be true (from reader)")
	}
	if !merged.CanCreateLinks {
		t.Error("merged: CanCreateLinks should be true (from creator)")
	}
	if !merged.CanEditOwnLinks {
		t.Error("merged: CanEditOwnLinks should be true (from creator)")
	}
	if !merged.CanViewOwnStats {
		t.Error("merged: CanViewOwnStats should be true (from reader)")
	}
	if merged.CanManageUsers {
		t.Error("merged: CanManageUsers should be false")
	}
	if merged.CanViewAllLinks {
		t.Error("merged: CanViewAllLinks should be false")
	}
}

func TestPermissionsCache_GetMerged_SingleRole(t *testing.T) {
	roles := []domain.RolePermissions{
		{Role: "editor", CanCreateLinks: true, CanEditOwnLinks: true},
	}
	cache := newCache(t, roles, "admin")

	merged := cache.GetMerged([]string{"editor"})
	direct := cache.Get("editor")

	if merged.CanCreateLinks != direct.CanCreateLinks {
		t.Error("GetMerged(single) should equal Get")
	}
	if merged.CanEditOwnLinks != direct.CanEditOwnLinks {
		t.Error("GetMerged(single) should equal Get")
	}
}

func TestPermissionsCache_GetMerged_Empty(t *testing.T) {
	cache := newCache(t, nil, "admin")
	merged := cache.GetMerged([]string{})

	if merged.CanViewOwnLinks {
		t.Error("empty roles: CanViewOwnLinks should be false")
	}
	if merged.CanCreateLinks {
		t.Error("empty roles: CanCreateLinks should be false")
	}
}

// --- GetAll ---

func TestPermissionsCache_GetAll_Empty(t *testing.T) {
	cache := newCache(t, nil, "admin")
	all := cache.GetAll()
	if len(all) != 0 {
		t.Errorf("expected 0 roles, got %d", len(all))
	}
}

func TestPermissionsCache_GetAll_AfterLoad(t *testing.T) {
	roles := []domain.RolePermissions{
		{Role: "editor", CanCreateLinks: true},
		{Role: "viewer", CanViewOwnLinks: true},
	}
	cache := newCache(t, roles, "admin")
	all := cache.GetAll()
	if len(all) != 2 {
		t.Errorf("expected 2 roles, got %d", len(all))
	}
}

func TestPermissionsCache_GetAll_AfterSet(t *testing.T) {
	cache := newCache(t, nil, "admin")
	cache.Set(domain.RolePermissions{Role: "editor", CanCreateLinks: true})
	cache.Set(domain.RolePermissions{Role: "viewer", CanViewOwnLinks: true})

	all := cache.GetAll()
	if len(all) != 2 {
		t.Errorf("expected 2 roles after Set, got %d", len(all))
	}
}

// --- Вспомогательные функции ---

// newStubRepo возвращает service.RolesRepo-совместимый stub без реальной БД.
func newStubRepo(rows []domain.RolePermissions) service.RolesRepo {
	return &stubRepoImpl{data: rows}
}

type stubRepoImpl struct {
	data []domain.RolePermissions
}

func (r *stubRepoImpl) GetAll(_ context.Context) ([]domain.RolePermissions, error) {
	return r.data, nil
}

func (r *stubRepoImpl) Upsert(_ context.Context, p *domain.RolePermissions) error {
	for i, d := range r.data {
		if d.Role == p.Role {
			r.data[i] = *p
			return nil
		}
	}
	r.data = append(r.data, *p)
	return nil
}
