package test

import (
	"context"
	"testing"

	"unified-backend/internal/config"
	"unified-backend/internal/domain"
	"unified-backend/internal/service"
	"unified-backend/internal/shlink"
)

func newShlinkService(slugPrefixEnabled bool) *service.ShlinkService {
	cfg := &config.Config{
		UserSlugPrefixEnabled:    slugPrefixEnabled,
		UserTagInternalIdEnabled: false,
		ShlinkURL:                "http://shlink-api:8080",
	}
	cli := shlink.NewClient(cfg.ShlinkURL)
	return service.NewShlinkService(cli, cfg)
}

// TestEnforceSlugPrefix_AdminBypass — для admin prefix не применяется
func TestEnforceSlugPrefix_AdminBypass(t *testing.T) {
	svc := newShlinkService(true)
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

// TestEnforceSlugPrefix_UserNoPrefix — feature enabled, нет prefix → ошибка
func TestEnforceSlugPrefix_UserNoPrefix(t *testing.T) {
	svc := newShlinkService(true)
	user := &domain.User{Role: domain.RoleUser, SlugPrefix: ""}
	slug := "my-slug"

	_, err := svc.EnforceSlugPrefix(context.Background(), user, &slug)
	if err == nil {
		t.Error("expected error when user has no slug prefix")
	}
}

// TestEnforceSlugPrefix_UserCorrectPrefix — slug с правильным prefix → OK
func TestEnforceSlugPrefix_UserCorrectPrefix(t *testing.T) {
	svc := newShlinkService(true)
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

// TestEnforceSlugPrefix_UserWrongPrefix — slug без prefix → ошибка
func TestEnforceSlugPrefix_UserWrongPrefix(t *testing.T) {
	svc := newShlinkService(true)
	user := &domain.User{Role: domain.RoleUser, SlugPrefix: "u1-"}
	slug := "admin-link"

	_, err := svc.EnforceSlugPrefix(context.Background(), user, &slug)
	if err == nil {
		t.Error("expected error for slug without correct prefix")
	}
}

// TestEnforceSlugPrefix_FeatureDisabled — feature выключен → slug не трогается
func TestEnforceSlugPrefix_FeatureDisabled(t *testing.T) {
	svc := newShlinkService(false)
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

// TestEnforceSlugPrefix_UserNilSlug — nil slug + prefix → возвращает prefix
func TestEnforceSlugPrefix_UserNilSlug(t *testing.T) {
	svc := newShlinkService(true)
	user := &domain.User{Role: domain.RoleUser, SlugPrefix: "u2-"}

	result, err := svc.EnforceSlugPrefix(context.Background(), user, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "u2-" {
		t.Errorf("expected prefix %q, got %q", "u2-", result)
	}
}

// TestFilterShortURLsByUser — фильтрация по prefix
func TestFilterShortURLsByUser(t *testing.T) {
	svc := newShlinkService(true)
	user := &domain.User{Role: domain.RoleUser, SlugPrefix: "u1-"}

	urls := []shlink.ShortURL{
		{ShortCode: "u1-abc"},
		{ShortCode: "u1-xyz"},
		{ShortCode: "u2-abc"},
		{ShortCode: "random"},
	}

	filtered := svc.FilterShortURLsByUser(urls, user)

	if len(filtered) != 2 {
		t.Errorf("expected 2 filtered URLs, got %d", len(filtered))
	}
	for _, u := range filtered {
		if u.ShortCode[:3] != "u1-" {
			t.Errorf("expected prefix u1-, got %s", u.ShortCode)
		}
	}
}

// TestFilterShortURLsByUser_AdminGetAll — admin видит все
func TestFilterShortURLsByUser_AdminGetAll(t *testing.T) {
	svc := newShlinkService(true)
	admin := &domain.User{Role: domain.RoleAdmin, SlugPrefix: ""}

	urls := []shlink.ShortURL{
		{ShortCode: "u1-abc"},
		{ShortCode: "u2-xyz"},
		{ShortCode: "random"},
	}

	result := svc.FilterShortURLsByUser(urls, admin)
	if len(result) != 3 {
		t.Errorf("admin should see all URLs, got %d", len(result))
	}
}

// TestComputePermissions_Admin — admin получает все права
func TestComputePermissions_Admin(t *testing.T) {
	user := &domain.User{Role: domain.RoleAdmin}
	perms := user.ComputePermissions()

	if !perms.CanViewAuditLogs {
		t.Error("admin should canViewAuditLogs")
	}
	if !perms.CanManageUsers {
		t.Error("admin should canManageUsers")
	}
}

// TestComputePermissions_User — user не получает admin-права
func TestComputePermissions_User(t *testing.T) {
	user := &domain.User{Role: domain.RoleUser}
	perms := user.ComputePermissions()

	if perms.CanViewAuditLogs {
		t.Error("user should NOT canViewAuditLogs")
	}
	if perms.CanManageUsers {
		t.Error("user should NOT canManageUsers")
	}
	if !perms.CanCreateShortURL {
		t.Error("user SHOULD canCreateShortUrl")
	}
}
