package test

import (
	"context"
	"strings"
	"testing"

	"unified-backend/internal/config"
	"unified-backend/internal/domain"
	"unified-backend/internal/service"
	"unified-backend/internal/shlink"
)

// ── helpers ──────────────────────────────────────────────────────────────────

type stubRolesRepoSvc struct{ data []domain.RolePermissions }

func (s *stubRolesRepoSvc) GetAll(_ context.Context) ([]domain.RolePermissions, error) {
	return s.data, nil
}

func (s *stubRolesRepoSvc) Upsert(_ context.Context, p *domain.RolePermissions) error {
	for i, d := range s.data {
		if d.Role == p.Role {
			s.data[i] = *p
			return nil
		}
	}
	s.data = append(s.data, *p)
	return nil
}

func newShlinkServiceWithPerms(perms []domain.RolePermissions, cfg *config.Config) *service.ShlinkService {
	if cfg.ShlinkURL == "" {
		cfg.ShlinkURL = "http://shlink-api:8080"
	}
	cache := service.NewPermissionsCache(&stubRolesRepoSvc{data: perms}, cfg.AdminRole)
	_ = cache.Load(context.Background())
	cli := shlink.NewClient(cfg.ShlinkURL)
	return service.NewShlinkService(cli, cfg, cache)
}

func newShlinkService(p domain.RolePermissions) *service.ShlinkService {
	return newShlinkServiceWithPerms([]domain.RolePermissions{p}, &config.Config{
		AdminRole:             domain.RoleAdmin,
		UserSlugPrefixEnabled: false,
		UserCustomSlugEnabled: true,
	})
}

func newShlinkServiceFull(slugPrefixEnabled, userCustomSlugEnabled bool) *service.ShlinkService {
	return newShlinkServiceWithPerms(
		[]domain.RolePermissions{
			domain.DefaultAdminPermissions(domain.RoleAdmin),
			domain.DefaultUserPermissions(domain.RoleUser),
		},
		&config.Config{
			AdminRole:             domain.RoleAdmin,
			UserSlugPrefixEnabled: slugPrefixEnabled,
			UserCustomSlugEnabled: userCustomSlugEnabled,
		},
	)
}

// ── EnforceSlugPrefix ──────────────────────────────────────────────────────────────

func TestEnforceSlugPrefix_AdminBypass(t *testing.T) {
	svc := newShlinkServiceFull(true, true)
	admin := &domain.User{Role: domain.RoleAdmin, SlugPrefix: "adm-"}
	slug := "my-custom-slug"

	result, err := svc.EnforceSlugPrefix(context.Background(), admin, &slug)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != slug {
		t.Errorf("admin slug should be unchanged: expected %q, got %q", slug, result)
	}
}

func TestEnforceSlugPrefix_UserNoPrefix(t *testing.T) {
	svc := newShlinkServiceFull(true, true)
	user := &domain.User{Role: domain.RoleUser, SlugPrefix: ""}
	slug := "my-slug"

	_, err := svc.EnforceSlugPrefix(context.Background(), user, &slug)
	if err == nil {
		t.Error("expected error when user has no slug prefix")
	}
}

func TestEnforceSlugPrefix_UserCorrectPrefix(t *testing.T) {
	svc := newShlinkServiceFull(true, true)
	user := &domain.User{Role: domain.RoleUser, SlugPrefix: "u1-"}
	slug := "u1-mylink"

	result, err := svc.EnforceSlugPrefix(context.Background(), user, &slug)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "u1-mylink" {
		t.Errorf("expected %q, got %q", "u1-mylink", result)
	}
}

func TestEnforceSlugPrefix_UserWrongPrefix(t *testing.T) {
	svc := newShlinkServiceFull(true, true)
	user := &domain.User{Role: domain.RoleUser, SlugPrefix: "u1-"}
	slug := "admin-link"

	_, err := svc.EnforceSlugPrefix(context.Background(), user, &slug)
	if err == nil {
		t.Error("expected error for slug without correct prefix")
	}
}

func TestEnforceSlugPrefix_FeatureDisabled(t *testing.T) {
	svc := newShlinkServiceFull(false, true)
	user := &domain.User{Role: domain.RoleUser, SlugPrefix: "u1-"}
	slug := "any-slug"

	result, err := svc.EnforceSlugPrefix(context.Background(), user, &slug)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != slug {
		t.Errorf("when feature disabled, slug should be unchanged: got %q", result)
	}
}

func TestEnforceSlugPrefix_UserNilSlug(t *testing.T) {
	svc := newShlinkServiceFull(true, true)
	user := &domain.User{Role: domain.RoleUser, SlugPrefix: "u2-"}

	result, err := svc.EnforceSlugPrefix(context.Background(), user, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "u2-" {
		t.Errorf("expected prefix %q, got %q", "u2-", result)
	}
}

func TestEnforceSlugPrefix_UserCustomSlugFeatureDisabled(t *testing.T) {
	svc := newShlinkServiceFull(false, false)
	user := &domain.User{Role: domain.RoleUser, SlugPrefix: "u1-"}
	slug := "u1-mylink"

	_, err := svc.EnforceSlugPrefix(context.Background(), user, &slug)
	if err == nil {
		t.Fatal("expected error when UserCustomSlugEnabled=false")
	}
	if !strings.Contains(err.Error(), "custom slugs are disabled") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestEnforceSlugPrefix_AdminIgnoresFeatureFlag(t *testing.T) {
	svc := newShlinkServiceFull(false, false)
	admin := &domain.User{Role: domain.RoleAdmin, SlugPrefix: "adm-"}
	slug := "anything"

	result, err := svc.EnforceSlugPrefix(context.Background(), admin, &slug)
	if err != nil {
		t.Fatalf("admin should ignore UserCustomSlugEnabled=false, got error: %v", err)
	}
	if result != slug {
		t.Errorf("expected %q, got %q", slug, result)
	}
}

// ── DefaultPermissions ──────────────────────────────────────────────────────────────

func TestDefaultPermissions_Admin(t *testing.T) {
	perms := domain.DefaultAdminPermissions(domain.RoleAdmin)

	if !perms.CanViewAuditLogs {
		t.Error("admin should canViewAuditLogs")
	}
	if !perms.CanManageUsers {
		t.Error("admin should canManageUsers")
	}
}

func TestDefaultPermissions_User(t *testing.T) {
	perms := domain.DefaultUserPermissions(domain.RoleUser)

	if perms.CanViewAuditLogs {
		t.Error("user should NOT canViewAuditLogs")
	}
	if perms.CanManageUsers {
		t.Error("user should NOT canManageUsers")
	}
	if !perms.CanCreateLinks {
		t.Error("user SHOULD canCreateLinks")
	}
}
