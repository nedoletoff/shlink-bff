package test

import (
	"context"
	"testing"

	"unified-backend/internal/config"
	"unified-backend/internal/domain"
	"unified-backend/internal/shlink"
)

// rbacCase describes one RBAC scenario: permissions + expected outcomes.
type rbacCase struct {
	name  string
	perms domain.RolePermissions
	// FilterShortURLsByUser expectations
	ownedCodes   []string
	allURLs      []shlink.ShortURL
	wantURLCount int
	// CanModifyShortCodeByPerms expectations
	wantCanEditAll bool
	wantCanEditOwn bool
	wantCanDelAll  bool
	wantCanDelOwn  bool
}

var rbacMatrix = []rbacCase{
	{
		name:  "admin_full_access",
		perms: domain.DefaultAdminPermissions(domain.RoleAdmin),
		allURLs: []shlink.ShortURL{
			{ShortCode: "a"}, {ShortCode: "b"}, {ShortCode: "c"},
		},
		ownedCodes:   []string{"a"},
		wantURLCount: 3,
		wantCanEditAll: true, wantCanEditOwn: true,
		wantCanDelAll:  true, wantCanDelOwn:  true,
	},
	{
		name:  "user_own_only",
		perms: domain.DefaultUserPermissions(domain.RoleUser),
		allURLs: []shlink.ShortURL{
			{ShortCode: "u1-abc"}, {ShortCode: "u1-xyz"}, {ShortCode: "u2-foreign"},
		},
		ownedCodes:   []string{"u1-abc", "u1-xyz"},
		wantURLCount: 2,
		wantCanEditAll: false, wantCanEditOwn: true,
		wantCanDelAll:  false, wantCanDelOwn:  true,
	},
	{
		name: "viewer_read_only",
		perms: domain.RolePermissions{
			Role:            "viewer",
			CanViewAllLinks: true,
		},
		allURLs: []shlink.ShortURL{
			{ShortCode: "x"}, {ShortCode: "y"},
		},
		ownedCodes:   []string{"x"},
		wantURLCount: 2,
		wantCanEditAll: false, wantCanEditOwn: false,
		wantCanDelAll:  false, wantCanDelOwn:  false,
	},
	{
		name: "no_view_permission",
		perms: domain.RolePermissions{
			Role:           "creator_only",
			CanCreateLinks: true,
		},
		allURLs: []shlink.ShortURL{
			{ShortCode: "z"},
		},
		ownedCodes:   []string{"z"},
		wantURLCount: 0,
		wantCanEditAll: false, wantCanEditOwn: false,
		wantCanDelAll:  false, wantCanDelOwn:  false,
	},
	{
		name: "delete_all_no_edit",
		perms: domain.RolePermissions{
			Role:              "janitor",
			CanViewAllLinks:   true,
			CanDeleteAllLinks: true,
		},
		allURLs: []shlink.ShortURL{
			{ShortCode: "p"}, {ShortCode: "q"},
		},
		ownedCodes:   nil,
		wantURLCount: 2,
		wantCanEditAll: false, wantCanEditOwn: false,
		wantCanDelAll:  true, wantCanDelOwn:  false,
	},
	{
		name: "own_stats_only",
		perms: domain.RolePermissions{
			Role:            "stats_viewer",
			CanViewOwnLinks: true,
			CanViewOwnStats: true,
		},
		allURLs: []shlink.ShortURL{
			{ShortCode: "s1"}, {ShortCode: "s2"}, {ShortCode: "s3"},
		},
		ownedCodes:   []string{"s1"},
		wantURLCount: 1,
		wantCanEditAll: false, wantCanEditOwn: false,
		wantCanDelAll:  false, wantCanDelOwn:  false,
	},
}

func TestRBACMatrix(t *testing.T) {
	for _, tc := range rbacMatrix {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			svc := newShlinkService(tc.perms)
			user := &domain.User{Role: tc.perms.Role, Sub: "test-sub"}

			// FilterShortURLsByUser
			got := svc.FilterShortURLsByUser(tc.allURLs, user, ownedSet(tc.ownedCodes...))
			if len(got) != tc.wantURLCount {
				t.Errorf("FilterShortURLsByUser: want %d, got %d", tc.wantURLCount, len(got))
			}

			// CanModifyShortCodeByPerms — edit
			canAll, canOwn := svc.CanModifyShortCodeByPerms(user, false)
			if canAll != tc.wantCanEditAll {
				t.Errorf("CanEditAllLinks: want %v, got %v", tc.wantCanEditAll, canAll)
			}
			if canOwn != tc.wantCanEditOwn {
				t.Errorf("CanEditOwnLinks: want %v, got %v", tc.wantCanEditOwn, canOwn)
			}

			// CanModifyShortCodeByPerms — delete
			canAll, canOwn = svc.CanModifyShortCodeByPerms(user, true)
			if canAll != tc.wantCanDelAll {
				t.Errorf("CanDeleteAllLinks: want %v, got %v", tc.wantCanDelAll, canAll)
			}
			if canOwn != tc.wantCanDelOwn {
				t.Errorf("CanDeleteOwnLinks: want %v, got %v", tc.wantCanDelOwn, canOwn)
			}
		})
	}
}

// ── newShlinkServicePrefixed ──────────────────────────────────────────────────

// newShlinkServicePrefixed creates a service with UserSlugPrefixEnabled=true
// and UserCustomSlugEnabled=true, for tests that exercise prefix enforcement.
func newShlinkServicePrefixed(p domain.RolePermissions) *service.ShlinkService {
	return newShlinkServiceWithPerms([]domain.RolePermissions{p}, &config.Config{
		AdminRole:             domain.RoleAdmin,
		UserSlugPrefixEnabled: true,
		UserCustomSlugEnabled: true,
	})
}

// ── EnforceSlugPrefix RBAC ─────────────────────────────────────────────────

func TestEnforceSlugPrefix_AdminNoPrefix(t *testing.T) {
	svc := newShlinkServicePrefixed(domain.DefaultAdminPermissions(domain.RoleAdmin))
	admin := &domain.User{Role: domain.RoleAdmin, Sub: "admin1", SlugPrefix: "adm-"}

	result, err := svc.EnforceSlugPrefix(context.TODO(), admin, strPtr("my-custom-slug"))
	if err != nil {
		t.Fatalf("admin: unexpected error: %v", err)
	}
	if result != "my-custom-slug" {
		t.Errorf("admin: slug must pass through unchanged, got %q", result)
	}
}

func TestEnforceSlugPrefix_UserPrefixEnforced(t *testing.T) {
	// UserSlugPrefixEnabled=true: prefix must be validated.
	svc := newShlinkServicePrefixed(domain.DefaultUserPermissions(domain.RoleUser))
	user := &domain.User{Role: domain.RoleUser, Sub: "u1", SlugPrefix: "u1-"}

	// slug without prefix — must be rejected
	_, err := svc.EnforceSlugPrefix(context.TODO(), user, strPtr("no-prefix"))
	if err == nil {
		t.Error("expected error when slug doesn't start with user prefix")
	}

	// slug with correct prefix — must pass
	result, err := svc.EnforceSlugPrefix(context.TODO(), user, strPtr("u1-link"))
	if err != nil {
		t.Fatalf("valid slug: unexpected error: %v", err)
	}
	if result != "u1-link" {
		t.Errorf("want u1-link, got %q", result)
	}
}

func TestEnforceSlugPrefix_NoCreatePermission(t *testing.T) {
	p := domain.RolePermissions{
		Role:            "viewer",
		CanViewAllLinks: true,
	}
	svc := newShlinkServicePrefixed(p)
	user := &domain.User{Role: "viewer", Sub: "v1"}

	_, err := svc.EnforceSlugPrefix(context.TODO(), user, nil)
	if err == nil {
		t.Error("viewer with CanCreateLinks=false must be denied")
	}
}

func TestEnforceSlugPrefix_NoSlugAndMustProvide(t *testing.T) {
	p := domain.RolePermissions{
		Role:                    "strictuser",
		CanCreateLinks:          true,
		CanCreateWithCustomSlug: true,
		// CanCreateWithoutSlug = false
	}
	svc := newShlinkServicePrefixed(p)
	user := &domain.User{Role: "strictuser", Sub: "s1", SlugPrefix: ""}

	_, err := svc.EnforceSlugPrefix(context.TODO(), user, nil)
	if err == nil {
		t.Error("role without CanCreateWithoutSlug must be denied when no slug provided")
	}
}

func strPtr(s string) *string { return &s }
