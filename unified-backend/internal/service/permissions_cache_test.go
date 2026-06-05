package service_test

import (
	"context"
	"sync"
	"testing"

	"unified-backend/internal/domain"
	"unified-backend/internal/service"
)

// ── stub repo ────────────────────────────────────────────────────────────────

type stubRolesRepo struct {
	data []domain.RolePermissions
	err  error
}

func (s *stubRolesRepo) GetAll(_ context.Context) ([]domain.RolePermissions, error) {
	return s.data, s.err
}
func (s *stubRolesRepo) Upsert(_ context.Context, p *domain.RolePermissions) error {
	for i, r := range s.data {
		if r.Role == p.Role {
			s.data[i] = *p
			return nil
		}
	}
	s.data = append(s.data, *p)
	return nil
}

// ── Load ─────────────────────────────────────────────────────────────────────

func TestLoad_PopulatesCache(t *testing.T) {
	repo := &stubRolesRepo{
		data: []domain.RolePermissions{
			{Role: "editor", CanCreateLinks: true},
			{Role: "viewer", CanViewOwnLinks: true},
		},
	}
	c := service.NewPermissionsCache(repo, "admin")
	if err := c.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	if p := c.Get("editor"); !p.CanCreateLinks {
		t.Error("editor should have CanCreateLinks")
	}
	if p := c.Get("viewer"); !p.CanViewOwnLinks {
		t.Error("viewer should have CanViewOwnLinks")
	}
}

func TestLoad_RepoError(t *testing.T) {
	repo := &stubRolesRepo{err: errStub}
	c := service.NewPermissionsCache(repo, "admin")
	if err := c.Load(context.Background()); err == nil {
		t.Error("expected error from Load")
	}
}

var errStub = &stubErr{"stub error"}

type stubErr struct{ msg string }

func (e *stubErr) Error() string { return e.msg }

// ── Get ──────────────────────────────────────────────────────────────────────

func TestGet_KnownRole(t *testing.T) {
	repo := &stubRolesRepo{
		data: []domain.RolePermissions{{Role: "editor", CanCreateLinks: true}},
	}
	c := service.NewPermissionsCache(repo, "admin")
	_ = c.Load(context.Background())
	p := c.Get("editor")
	if p.Role != "editor" || !p.CanCreateLinks {
		t.Errorf("unexpected: %+v", p)
	}
}

func TestGet_AdminFallback(t *testing.T) {
	c := service.NewPermissionsCache(&stubRolesRepo{}, "admin")
	_ = c.Load(context.Background())
	p := c.Get("admin")
	// admin not in DB → DefaultAdminPermissions
	if !p.CanManageUsers || !p.CanCreateLinks {
		t.Errorf("admin fallback missing flags: %+v", p)
	}
}

func TestGet_UnknownRoleFallback(t *testing.T) {
	c := service.NewPermissionsCache(&stubRolesRepo{}, "admin")
	_ = c.Load(context.Background())
	p := c.Get("ghost")
	// unknown → DefaultUserPermissions
	if p.CanManageUsers || p.CanViewAllLinks {
		t.Errorf("ghost should have minimal perms: %+v", p)
	}
	if !p.CanCreateLinks {
		t.Errorf("ghost should have CanCreateLinks (default user): %+v", p)
	}
}

// ── Set ──────────────────────────────────────────────────────────────────────

func TestSet_UpdatesCache(t *testing.T) {
	repo := &stubRolesRepo{
		data: []domain.RolePermissions{{Role: "editor", CanCreateLinks: false}},
	}
	c := service.NewPermissionsCache(repo, "admin")
	_ = c.Load(context.Background())

	c.Set(domain.RolePermissions{Role: "editor", CanCreateLinks: true, CanViewAllLinks: true})
	p := c.Get("editor")
	if !p.CanCreateLinks || !p.CanViewAllLinks {
		t.Errorf("cache not updated: %+v", p)
	}
}

// ── GetMerged ─────────────────────────────────────────────────────────────────

func TestGetMerged_ORSemantics(t *testing.T) {
	repo := &stubRolesRepo{
		data: []domain.RolePermissions{
			{Role: "editor", CanCreateLinks: true, CanViewOwnLinks: true},
			{Role: "moderator", CanDeleteAllLinks: true, CanViewOwnLinks: true},
		},
	}
	c := service.NewPermissionsCache(repo, "admin")
	_ = c.Load(context.Background())

	p := c.GetMerged([]string{"editor", "moderator"})
	if !p.CanCreateLinks {
		t.Error("merged should have CanCreateLinks from editor")
	}
	if !p.CanDeleteAllLinks {
		t.Error("merged should have CanDeleteAllLinks from moderator")
	}
	if p.Role != "" {
		t.Errorf("merged role should be empty, got %q", p.Role)
	}
}

func TestGetMerged_SingleRole(t *testing.T) {
	repo := &stubRolesRepo{
		data: []domain.RolePermissions{{Role: "editor", CanCreateLinks: true}},
	}
	c := service.NewPermissionsCache(repo, "admin")
	_ = c.Load(context.Background())

	p := c.GetMerged([]string{"editor"})
	if !p.CanCreateLinks {
		t.Error("single role merge should pass through permissions")
	}
}

func TestGetMerged_Empty(t *testing.T) {
	c := service.NewPermissionsCache(&stubRolesRepo{}, "admin")
	_ = c.Load(context.Background())

	p := c.GetMerged(nil)
	if p.CanCreateLinks || p.CanManageUsers {
		t.Error("empty roles should return zero permissions")
	}
}

// ── GetAll ────────────────────────────────────────────────────────────────────

func TestGetAll_ReturnsSnapshot(t *testing.T) {
	repo := &stubRolesRepo{
		data: []domain.RolePermissions{
			{Role: "a"}, {Role: "b"}, {Role: "c"},
		},
	}
	c := service.NewPermissionsCache(repo, "admin")
	_ = c.Load(context.Background())

	all := c.GetAll()
	if len(all) != 3 {
		t.Errorf("expected 3 roles, got %d", len(all))
	}
}

// ── Concurrency ───────────────────────────────────────────────────────────────

func TestCache_Concurrent(t *testing.T) {
	repo := &stubRolesRepo{
		data: []domain.RolePermissions{{Role: "editor", CanCreateLinks: true}},
	}
	c := service.NewPermissionsCache(repo, "admin")
	_ = c.Load(context.Background())

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			c.Get("editor")
		}()
		go func() {
			defer wg.Done()
			c.Set(domain.RolePermissions{Role: "editor", CanCreateLinks: true})
		}()
	}
	wg.Wait()
}
