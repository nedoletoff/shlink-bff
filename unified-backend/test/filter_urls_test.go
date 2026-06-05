package test

import (
	"testing"

	"unified-backend/internal/domain"
	"unified-backend/internal/shlink"
)

// ── helpers ────────────────────────────────────────────────────────────────

// ownedSet builds the map[string]struct{} that FilterShortURLsByUser expects.
func ownedSet(codes ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(codes))
	for _, c := range codes {
		m[c] = struct{}{}
	}
	return m
}

// makeAdminPerms returns full RolePermissions for admin-like role.
func makeAdminPerms() domain.RolePermissions {
	return domain.DefaultAdminPermissions(domain.RoleAdmin)
}

// makeUserPerms returns minimal user permissions with view/edit/delete own.
func makeUserPerms() domain.RolePermissions {
	return domain.DefaultUserPermissions(domain.RoleUser)
}

// ── FilterShortURLsByUser ──────────────────────────────────────────────────

func TestFilterShortURLsByUser_AdminSeesAll(t *testing.T) {
	svc := newShlinkService(makeAdminPerms())
	admin := &domain.User{Role: domain.RoleAdmin, Sub: "admin1"}

	urls := []shlink.ShortURL{
		{ShortCode: "aaa"},
		{ShortCode: "bbb"},
		{ShortCode: "ccc"},
	}
	// Admin has CanViewAllLinks=true → ownedCodes is ignored
	got := svc.FilterShortURLsByUser(urls, admin, nil)
	if len(got) != 3 {
		t.Errorf("admin: want 3, got %d", len(got))
	}
}

func TestFilterShortURLsByUser_UserSeesOnlyOwned(t *testing.T) {
	svc := newShlinkService(makeUserPerms())
	user := &domain.User{Role: domain.RoleUser, Sub: "u1"}

	urls := []shlink.ShortURL{
		{ShortCode: "u1-abc"},
		{ShortCode: "u1-xyz"},
		{ShortCode: "u2-abc"},
		{ShortCode: "random"},
	}
	// Only u1-abc and u1-xyz are in ownership table
	got := svc.FilterShortURLsByUser(urls, user, ownedSet("u1-abc", "u1-xyz"))
	if len(got) != 2 {
		t.Errorf("user: want 2 owned urls, got %d", len(got))
	}
	for _, u := range got {
		if u.ShortCode != "u1-abc" && u.ShortCode != "u1-xyz" {
			t.Errorf("unexpected shortCode %q in result", u.ShortCode)
		}
	}
}

func TestFilterShortURLsByUser_EmptyOwnershipSet(t *testing.T) {
	svc := newShlinkService(makeUserPerms())
	user := &domain.User{Role: domain.RoleUser, Sub: "u1"}
	urls := []shlink.ShortURL{
		{ShortCode: "u1-abc"},
		{ShortCode: "u2-abc"},
	}
	// User has CanViewOwnLinks but no records in ownership table
	got := svc.FilterShortURLsByUser(urls, user, ownedSet())
	if len(got) != 0 {
		t.Errorf("empty ownership: want 0, got %d", len(got))
	}
}

func TestFilterShortURLsByUser_NoViewPermission(t *testing.T) {
	// Role with neither CanViewOwnLinks nor CanViewAllLinks
	p := domain.RolePermissions{Role: "restricted", CanCreateLinks: true}
	svc := newShlinkService(p)
	user := &domain.User{Role: "restricted", Sub: "u1"}

	urls := []shlink.ShortURL{{ShortCode: "any"}}
	got := svc.FilterShortURLsByUser(urls, user, ownedSet("any"))
	if len(got) != 0 {
		t.Errorf("no view perm: want 0, got %d", len(got))
	}
}

func TestFilterShortURLsByUser_EmptyInputList(t *testing.T) {
	svc := newShlinkService(makeUserPerms())
	user := &domain.User{Role: domain.RoleUser, Sub: "u1"}
	got := svc.FilterShortURLsByUser([]shlink.ShortURL{}, user, ownedSet("u1-abc"))
	if len(got) != 0 {
		t.Errorf("empty input: want 0, got %d", len(got))
	}
}

// ── CanModifyShortCodeByPerms ──────────────────────────────────────────────

func TestCanModifyShortCodeByPerms_AdminCanAll(t *testing.T) {
	svc := newShlinkService(makeAdminPerms())
	admin := &domain.User{Role: domain.RoleAdmin, Sub: "admin1"}

	canAll, canOwn := svc.CanModifyShortCodeByPerms(admin, false)
	if !canAll {
		t.Error("admin: CanEditAllLinks must be true")
	}
	if !canOwn {
		t.Error("admin: CanEditOwnLinks must be true")
	}

	canAll, canOwn = svc.CanModifyShortCodeByPerms(admin, true)
	if !canAll {
		t.Error("admin: CanDeleteAllLinks must be true")
	}
	if !canOwn {
		t.Error("admin: CanDeleteOwnLinks must be true")
	}
}

func TestCanModifyShortCodeByPerms_UserCanOwnOnly(t *testing.T) {
	svc := newShlinkService(makeUserPerms())
	user := &domain.User{Role: domain.RoleUser, Sub: "u1"}

	canAll, canOwn := svc.CanModifyShortCodeByPerms(user, false)
	if canAll {
		t.Error("user: CanEditAllLinks must be false")
	}
	if !canOwn {
		t.Error("user: CanEditOwnLinks must be true")
	}

	canAll, canOwn = svc.CanModifyShortCodeByPerms(user, true)
	if canAll {
		t.Error("user: CanDeleteAllLinks must be false")
	}
	if !canOwn {
		t.Error("user: CanDeleteOwnLinks must be true")
	}
}

func TestCanModifyShortCodeByPerms_ReadOnlyRole(t *testing.T) {
	p := domain.RolePermissions{
		Role:            "viewer",
		CanViewAllLinks: true,
		CanViewOwnLinks: true,
	}
	svc := newShlinkService(p)
	user := &domain.User{Role: "viewer", Sub: "v1"}

	canAll, canOwn := svc.CanModifyShortCodeByPerms(user, false)
	if canAll || canOwn {
		t.Error("viewer: must have no edit permissions")
	}

	canAll, canOwn = svc.CanModifyShortCodeByPerms(user, true)
	if canAll || canOwn {
		t.Error("viewer: must have no delete permissions")
	}
}

func TestCanModifyShortCodeByPerms_DeleteOnlyRole(t *testing.T) {
	// Edge case: role has delete but not edit
	p := domain.RolePermissions{
		Role:              "pruner",
		CanViewOwnLinks:   true,
		CanDeleteOwnLinks: true,
	}
	svc := newShlinkService(p)
	user := &domain.User{Role: "pruner", Sub: "p1"}

	canAll, canOwn := svc.CanModifyShortCodeByPerms(user, false) // edit
	if canAll || canOwn {
		t.Error("pruner: no edit perms expected")
	}

	_, canOwn = svc.CanModifyShortCodeByPerms(user, true) // delete
	if !canOwn {
		t.Error("pruner: CanDeleteOwnLinks expected true")
	}
}
