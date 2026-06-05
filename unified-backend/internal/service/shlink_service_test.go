package service_test

import (
	"context"
	"testing"

	"unified-backend/internal/config"
	"unified-backend/internal/domain"
	"unified-backend/internal/service"
	"unified-backend/internal/shlink"
)

// stubRolesRepo реализует service.RolePermissionsReader без БД.
type stubRolesRepo struct {
	data []domain.RolePermissions
}

func (s *stubRolesRepo) GetAll(_ context.Context) ([]domain.RolePermissions, error) {
	return s.data, nil
}

func newTestService(perms []domain.RolePermissions, cfg *config.Config) *service.ShlinkService {
	cache := service.NewPermissionsCache(&stubRolesRepo{data: perms}, cfg.AdminRole)
	_ = cache.Load(context.Background())
	if cfg.ShlinkURL == "" {
		cfg.ShlinkURL = "http://localhost"
	}
	return service.NewShlinkService(nil, cfg, cache)
}

func defaultCfg() *config.Config {
	return &config.Config{
		AdminRole:             "admin",
		ShlinkURL:             "http://localhost",
		UserSlugPrefixEnabled: false,
		UserCustomSlugEnabled: true,
	}
}

// ─── FilterShortURLsByUser ────────────────────────────────────────────────────

func TestFilterShortURLsByUser_CanViewAll_ReturnsAll(t *testing.T) {
	urls := []shlink.ShortURL{
		{ShortCode: "aaa"},
		{ShortCode: "bbb"},
	}
	svc := newTestService([]domain.RolePermissions{
		{Role: "admin", CanViewAllLinks: true, CanViewOwnLinks: true},
	}, defaultCfg())

	user := &domain.User{Sub: "u1", Role: "admin"}
	result := svc.FilterShortURLsByUser(urls, user, nil)
	if len(result) != 2 {
		t.Errorf("admin should see all 2 urls, got %d", len(result))
	}
}

func TestFilterShortURLsByUser_NoViewPerm_ReturnsEmpty(t *testing.T) {
	urls := []shlink.ShortURL{
		{ShortCode: "aaa"},
	}
	svc := newTestService([]domain.RolePermissions{
		{Role: "guest", CanViewAllLinks: false, CanViewOwnLinks: false},
	}, defaultCfg())

	user := &domain.User{Sub: "u1", Role: "guest"}
	result := svc.FilterShortURLsByUser(urls, user, nil)
	if len(result) != 0 {
		t.Errorf("guest without view perm should see 0 urls, got %d", len(result))
	}
}

func TestFilterShortURLsByUser_OwnLinks_FiltersByOwnership(t *testing.T) {
	urls := []shlink.ShortURL{
		{ShortCode: "aaa"},
		{ShortCode: "bbb"},
		{ShortCode: "ccc"},
	}
	svc := newTestService([]domain.RolePermissions{
		{Role: "user", CanViewOwnLinks: true, CanViewAllLinks: false},
	}, defaultCfg())

	user := &domain.User{Sub: "u1", Role: "user"}
	ownedCodes := map[string]struct{}{
		"aaa": {},
		"ccc": {},
	}
	result := svc.FilterShortURLsByUser(urls, user, ownedCodes)
	if len(result) != 2 {
		t.Errorf("user should see 2 owned urls, got %d", len(result))
	}
	for _, u := range result {
		if u.ShortCode == "bbb" {
			t.Error("bbb should be filtered out")
		}
	}
}

func TestFilterShortURLsByUser_OwnLinks_EmptyOwnerSet(t *testing.T) {
	urls := []shlink.ShortURL{
		{ShortCode: "aaa"},
	}
	svc := newTestService([]domain.RolePermissions{
		{Role: "user", CanViewOwnLinks: true, CanViewAllLinks: false},
	}, defaultCfg())

	user := &domain.User{Sub: "u1", Role: "user"}
	result := svc.FilterShortURLsByUser(urls, user, map[string]struct{}{})
	if len(result) != 0 {
		t.Errorf("user with no owned codes should see 0 urls, got %d", len(result))
	}
}

// ─── CanModifyShortCodeByPerms ────────────────────────────────────────────────

func TestCanModifyShortCodeByPerms_Admin_CanAll(t *testing.T) {
	svc := newTestService([]domain.RolePermissions{
		{Role: "admin", CanEditAllLinks: true, CanDeleteAllLinks: true},
	}, defaultCfg())

	user := &domain.User{Sub: "u1", Role: "admin"}

	canAll, _ := svc.CanModifyShortCodeByPerms(user, false)
	if !canAll {
		t.Error("admin should have canAll edit")
	}
	canAll, _ = svc.CanModifyShortCodeByPerms(user, true)
	if !canAll {
		t.Error("admin should have canAll delete")
	}
}

func TestCanModifyShortCodeByPerms_UserCanOwnOnly(t *testing.T) {
	svc := newTestService([]domain.RolePermissions{
		{Role: "user", CanEditOwnLinks: true, CanEditAllLinks: false, CanDeleteOwnLinks: true, CanDeleteAllLinks: false},
	}, defaultCfg())

	user := &domain.User{Sub: "u1", Role: "user"}

	canAll, canOwn := svc.CanModifyShortCodeByPerms(user, false)
	if canAll {
		t.Error("user should NOT have canAll edit")
	}
	if !canOwn {
		t.Error("user should have canOwn edit")
	}

	canAll, canOwn = svc.CanModifyShortCodeByPerms(user, true)
	if canAll {
		t.Error("user should NOT have canAll delete")
	}
	if !canOwn {
		t.Error("user should have canOwn delete")
	}
}

func TestCanModifyShortCodeByPerms_GuestNoPerm(t *testing.T) {
	svc := newTestService([]domain.RolePermissions{
		{Role: "guest"},
	}, defaultCfg())

	user := &domain.User{Sub: "u1", Role: "guest"}
	canAll, canOwn := svc.CanModifyShortCodeByPerms(user, false)
	if canAll || canOwn {
		t.Error("guest should have no modify permissions")
	}
}

// ─── EnforceSlugPrefix ────────────────────────────────────────────────────────

func TestEnforceSlugPrefix_NoCreatePerm_Error(t *testing.T) {
	svc := newTestService([]domain.RolePermissions{
		{Role: "readonly", CanCreateLinks: false},
	}, defaultCfg())

	user := &domain.User{Sub: "u1", Role: "readonly"}
	_, err := svc.EnforceSlugPrefix(context.Background(), user, nil)
	if err == nil {
		t.Error("expected error for role without CanCreateLinks")
	}
}

func TestEnforceSlugPrefix_NoSlugPrefixDisabled_ReturnsEmpty(t *testing.T) {
	cfg := defaultCfg()
	cfg.UserSlugPrefixEnabled = false
	svc := newTestService([]domain.RolePermissions{
		{Role: "user", CanCreateLinks: true, CanCreateWithoutSlug: true},
	}, cfg)

	user := &domain.User{Sub: "u1", Role: "user", SlugPrefix: "u-"}
	enforced, err := svc.EnforceSlugPrefix(context.Background(), user, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if enforced != "" {
		t.Errorf("expected empty slug when prefix disabled, got %q", enforced)
	}
}

func TestEnforceSlugPrefix_PrefixEnabled_ForcesPrefixOnEmpty(t *testing.T) {
	cfg := defaultCfg()
	cfg.UserSlugPrefixEnabled = true
	svc := newTestService([]domain.RolePermissions{
		{Role: "user", CanCreateLinks: true, CanCreateWithoutSlug: true},
	}, cfg)

	user := &domain.User{Sub: "u1", Role: "user", SlugPrefix: "u-"}
	enforced, err := svc.EnforceSlugPrefix(context.Background(), user, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if enforced != "u-" {
		t.Errorf("expected slug prefix 'u-', got %q", enforced)
	}
}

func TestEnforceSlugPrefix_PrefixEnabled_CustomSlugWithoutPrefix_Error(t *testing.T) {
	cfg := defaultCfg()
	cfg.UserSlugPrefixEnabled = true
	cfg.UserCustomSlugEnabled = true
	svc := newTestService([]domain.RolePermissions{
		{Role: "user", CanCreateLinks: true, CanCreateWithCustomSlug: true, CanCreateWithoutSlug: true},
	}, cfg)

	user := &domain.User{Sub: "u1", Role: "user", SlugPrefix: "u-"}
	slug := "wrongprefix-slug"
	_, err := svc.EnforceSlugPrefix(context.Background(), user, &slug)
	if err == nil {
		t.Error("expected error when slug doesn't start with prefix")
	}
}

func TestEnforceSlugPrefix_PrefixEnabled_CustomSlugWithPrefix_OK(t *testing.T) {
	cfg := defaultCfg()
	cfg.UserSlugPrefixEnabled = true
	cfg.UserCustomSlugEnabled = true
	svc := newTestService([]domain.RolePermissions{
		{Role: "user", CanCreateLinks: true, CanCreateWithCustomSlug: true, CanCreateWithoutSlug: true},
	}, cfg)

	user := &domain.User{Sub: "u1", Role: "user", SlugPrefix: "u-"}
	slug := "u-mylink"
	enforced, err := svc.EnforceSlugPrefix(context.Background(), user, &slug)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if enforced != slug {
		t.Errorf("expected %q, got %q", slug, enforced)
	}
}

func TestEnforceSlugPrefix_CustomSlugDisabled_NonAdmin_Error(t *testing.T) {
	cfg := defaultCfg()
	cfg.UserCustomSlugEnabled = false
	svc := newTestService([]domain.RolePermissions{
		{Role: "user", CanCreateLinks: true, CanCreateWithCustomSlug: true, CanCreateWithoutSlug: true},
	}, cfg)

	user := &domain.User{Sub: "u1", Role: "user"}
	slug := "myslug"
	_, err := svc.EnforceSlugPrefix(context.Background(), user, &slug)
	if err == nil {
		t.Error("expected error when UserCustomSlugEnabled=false for non-admin")
	}
}
