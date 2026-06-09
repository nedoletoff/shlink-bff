package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"unified-backend/internal/domain"
	"unified-backend/internal/middleware"
	"unified-backend/internal/service"
)

// stubRolesRepo повторно определён здесь для пакета middleware_test.
type stubMWRolesRepo struct {
	data []domain.RolePermissions
}

func (s *stubMWRolesRepo) GetAll(_ context.Context) ([]domain.RolePermissions, error) {
	return s.data, nil
}
func (s *stubMWRolesRepo) Upsert(_ context.Context, p *domain.RolePermissions) error {
	return nil
}

func newCache(perms ...domain.RolePermissions) *service.PermissionsCache {
	repo := &stubMWRolesRepo{data: perms}
	c := service.NewPermissionsCache(repo, "admin")
	_ = c.Load(context.Background())
	return c
}

func userCtx(u *domain.User) context.Context {
	return middleware.WithUser(context.Background(), u)
}

// ── RequirePermission ─────────────────────────────────────────────────────

func TestRequirePermission_HasFlag_Passes(t *testing.T) {
	cache := newCache(domain.RolePermissions{Role: "editor", CanCreateLinks: true})
	mw := middleware.RequirePermission(cache, func(p domain.RolePermissions) bool {
		return p.CanCreateLinks
	}, nil)
	h := mw(okHandler())

	u := &domain.User{Sub: "s1", Role: "editor"}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(userCtx(u))
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("editor with flag should pass, got %d", rec.Code)
	}
}

func TestRequirePermission_MissingFlag_Forbidden(t *testing.T) {
	cache := newCache(domain.RolePermissions{Role: "viewer", CanCreateLinks: false})
	mw := middleware.RequirePermission(cache, func(p domain.RolePermissions) bool {
		return p.CanCreateLinks
	}, nil)
	h := mw(okHandler())

	u := &domain.User{Sub: "s2", Role: "viewer"}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(userCtx(u))
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("viewer without flag should be forbidden, got %d", rec.Code)
	}
}

func TestRequirePermission_NoUser_Forbidden(t *testing.T) {
	cache := newCache()
	mw := middleware.RequirePermission(cache, func(p domain.RolePermissions) bool {
		return p.CanCreateLinks
	}, nil)
	h := mw(okHandler())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("no user should be forbidden, got %d", rec.Code)
	}
}

// ── RequirePermission + multi-role OR-semantics ────────────────────────────────

func TestRequirePermission_MultiRole_OR(t *testing.T) {
	cache := newCache(
		domain.RolePermissions{Role: "editor", CanCreateLinks: true},
		domain.RolePermissions{Role: "viewer", CanCreateLinks: false},
	)
	mw := middleware.RequirePermission(cache, func(p domain.RolePermissions) bool {
		return p.CanCreateLinks
	}, nil)
	h := mw(okHandler())

	u := &domain.User{Sub: "s3", Role: "viewer"}
	ctx := middleware.WithUser(context.Background(), u)
	ctx = middleware.WithRoles(ctx, []string{"viewer", "editor"})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("viewer+editor should pass via OR, got %d", rec.Code)
	}
}

// ── RequireRole ───────────────────────────────────────────────────────────────

func TestRequireRole_MatchingRole_Passes(t *testing.T) {
	mw := middleware.RequireRole("admin", nil)
	h := mw(okHandler())

	u := &domain.User{Sub: "s1", Role: "admin"}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil).WithContext(userCtx(u))
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("admin role should pass, got %d", rec.Code)
	}
}

func TestRequireRole_WrongRole_Forbidden(t *testing.T) {
	mw := middleware.RequireRole("admin", nil)
	h := mw(okHandler())

	u := &domain.User{Sub: "s2", Role: "user"}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil).WithContext(userCtx(u))
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("user role should be forbidden, got %d", rec.Code)
	}
}

func TestRequireRole_NoUser_Forbidden(t *testing.T) {
	mw := middleware.RequireRole("admin", nil)
	h := mw(okHandler())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("no user should be forbidden, got %d", rec.Code)
	}
}

func TestRequireRole_CaseInsensitive_Passes(t *testing.T) {
	mw := middleware.RequireRole("Admin", nil)
	h := mw(okHandler())

	u := &domain.User{Sub: "s4", Role: "ADMIN"}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil).WithContext(userCtx(u))
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("EqualFold: 'ADMIN' should match 'Admin', got %d", rec.Code)
	}
}
