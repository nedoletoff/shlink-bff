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
// Нужен только для NewPermissionsCache — без реального Postgres.
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

// ── AdminOnly ─────────────────────────────────────────────────────────────────

func TestAdminOnly_AdminRole_Passes(t *testing.T) {
	mw := middleware.AdminOnly("admin", nil)
	handler := mw(okHandler())

	u := &domain.User{Sub: "s1", Role: "admin"}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil).WithContext(userCtx(u))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("admin should pass, got %d", rec.Code)
	}
}

func TestAdminOnly_UserRole_Forbidden(t *testing.T) {
	mw := middleware.AdminOnly("admin", nil)
	handler := mw(okHandler())

	u := &domain.User{Sub: "s2", Role: "user"}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil).WithContext(userCtx(u))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("user should be forbidden, got %d", rec.Code)
	}
}

func TestAdminOnly_NoUser_Forbidden(t *testing.T) {
	mw := middleware.AdminOnly("admin", nil)
	handler := mw(okHandler())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("no user should be forbidden, got %d", rec.Code)
	}
}

func TestAdminOnly_EmptyRole_Forbidden(t *testing.T) {
	mw := middleware.AdminOnly("admin", nil)
	handler := mw(okHandler())

	u := &domain.User{Sub: "s3", Role: ""}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil).WithContext(userCtx(u))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("empty role should be forbidden, got %d", rec.Code)
	}
}

// ── EqualFold: регистр роли не важен ─────────────────────────────────────────

func TestAdminOnly_AdminRoleUpperCase_Passes(t *testing.T) {
	// ADMIN_ROLE=Admin в конфиге, Keycloak шлёт "admin" — должны совпасть
	mw := middleware.AdminOnly("Admin", nil)
	handler := mw(okHandler())

	u := &domain.User{Sub: "s4", Role: "admin"}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil).WithContext(userCtx(u))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("EqualFold: 'admin' should match 'Admin', got %d", rec.Code)
	}
}

func TestAdminOnly_AdminRoleMixedCase_Passes(t *testing.T) {
	// Keycloak даёт "ADMIN", конфиг задан "admin"
	mw := middleware.AdminOnly("admin", nil)
	handler := mw(okHandler())

	u := &domain.User{Sub: "s5", Role: "ADMIN"}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil).WithContext(userCtx(u))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("EqualFold: 'ADMIN' should match 'admin', got %d", rec.Code)
	}
}

func TestAdminOnly_AdminMixedBothSides_Passes(t *testing.T) {
	mw := middleware.AdminOnly("Admin", nil)
	handler := mw(okHandler())

	u := &domain.User{Sub: "s6", Role: "ADMIN"}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil).WithContext(userCtx(u))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("EqualFold: 'ADMIN' should match 'Admin', got %d", rec.Code)
	}
}

// ── RequirePermission ─────────────────────────────────────────────────────────

func TestRequirePermission_HasFlag_Passes(t *testing.T) {
	cache := newCache(domain.RolePermissions{Role: "editor", CanCreateLinks: true})
	mw := middleware.RequirePermission(cache, func(p domain.RolePermissions) bool {
		return p.CanCreateLinks
	}, nil)
	handler := mw(okHandler())

	u := &domain.User{Sub: "s1", Role: "editor"}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(userCtx(u))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("editor with flag should pass, got %d", rec.Code)
	}
}

func TestRequirePermission_MissingFlag_Forbidden(t *testing.T) {
	cache := newCache(domain.RolePermissions{Role: "viewer", CanCreateLinks: false})
	mw := middleware.RequirePermission(cache, func(p domain.RolePermissions) bool {
		return p.CanCreateLinks
	}, nil)
	handler := mw(okHandler())

	u := &domain.User{Sub: "s2", Role: "viewer"}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(userCtx(u))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("viewer without flag should be forbidden, got %d", rec.Code)
	}
}

func TestRequirePermission_NoUser_Forbidden(t *testing.T) {
	cache := newCache()
	mw := middleware.RequirePermission(cache, func(p domain.RolePermissions) bool {
		return p.CanCreateLinks
	}, nil)
	handler := mw(okHandler())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("no user should be forbidden, got %d", rec.Code)
	}
}

// ── RequirePermission + multi-role OR-semantics ───────────────────────────────

func TestRequirePermission_MultiRole_OR(t *testing.T) {
	cache := newCache(
		domain.RolePermissions{Role: "editor", CanCreateLinks: true},
		domain.RolePermissions{Role: "viewer", CanCreateLinks: false},
	)
	mw := middleware.RequirePermission(cache, func(p domain.RolePermissions) bool {
		return p.CanCreateLinks
	}, nil)
	handler := mw(okHandler())

	u := &domain.User{Sub: "s3", Role: "viewer"}
	ctx := middleware.WithUser(context.Background(), u)
	ctx = middleware.WithRoles(ctx, []string{"viewer", "editor"})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("viewer+editor should pass via OR, got %d", rec.Code)
	}
}
