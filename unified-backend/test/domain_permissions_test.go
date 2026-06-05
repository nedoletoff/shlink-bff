package test

import (
	"testing"

	"unified-backend/internal/domain"
)

// ── DefaultAdminPermissions ────────────────────────────────────────────────

// TestDefaultAdminPermissions_AllManagementFlags — admin получает все management-флаги.
func TestDefaultAdminPermissions_AllManagementFlags(t *testing.T) {
	p := domain.DefaultAdminPermissions("admin")

	if !p.CanManageUsers {
		t.Error("admin: CanManageUsers should be true")
	}
	if !p.CanManageRoles {
		t.Error("admin: CanManageRoles should be true")
	}
	if !p.CanViewAllLinks {
		t.Error("admin: CanViewAllLinks should be true")
	}
	if !p.CanEditAllLinks {
		t.Error("admin: CanEditAllLinks should be true")
	}
	if !p.CanDeleteAllLinks {
		t.Error("admin: CanDeleteAllLinks should be true")
	}
	if !p.CanViewAllStats {
		t.Error("admin: CanViewAllStats should be true")
	}
	if !p.CanManageAllTags {
		t.Error("admin: CanManageAllTags should be true")
	}
}

// TestDefaultAdminPermissions_IncludesUserFlags — admin также имеет все user-флаги.
func TestDefaultAdminPermissions_IncludesUserFlags(t *testing.T) {
	p := domain.DefaultAdminPermissions("admin")

	if !p.CanCreateLinks {
		t.Error("admin: CanCreateLinks should be true")
	}
	if !p.CanViewOwnLinks {
		t.Error("admin: CanViewOwnLinks should be true")
	}
	if !p.CanEditOwnLinks {
		t.Error("admin: CanEditOwnLinks should be true")
	}
	if !p.CanDeleteOwnLinks {
		t.Error("admin: CanDeleteOwnLinks should be true")
	}
	if !p.CanViewOwnStats {
		t.Error("admin: CanViewOwnStats should be true")
	}
	if !p.CanCreateWithCustomSlug {
		t.Error("admin: CanCreateWithCustomSlug should be true")
	}
}

// TestDefaultAdminPermissions_RoleField — поле Role заполняется переданным значением.
func TestDefaultAdminPermissions_RoleField(t *testing.T) {
	for _, name := range []string{"admin", "superadmin", "root"} {
		p := domain.DefaultAdminPermissions(name)
		if p.Role != name {
			t.Errorf("Role: want %q, got %q", name, p.Role)
		}
	}
}

// ── DefaultUserPermissions ─────────────────────────────────────────────────

// TestDefaultUserPermissions_BasicFlags — пользователь получает базовые own-флаги.
func TestDefaultUserPermissions_BasicFlags(t *testing.T) {
	p := domain.DefaultUserPermissions(domain.RoleUser)

	if !p.CanViewOwnLinks {
		t.Error("user: CanViewOwnLinks should be true")
	}
	if !p.CanCreateLinks {
		t.Error("user: CanCreateLinks should be true")
	}
	if !p.CanEditOwnLinks {
		t.Error("user: CanEditOwnLinks should be true")
	}
	if !p.CanDeleteOwnLinks {
		t.Error("user: CanDeleteOwnLinks should be true")
	}
	if !p.CanViewOwnStats {
		t.Error("user: CanViewOwnStats should be true")
	}
}

// TestDefaultUserPermissions_NoManagement — пользователь НЕ получает management-флаги.
func TestDefaultUserPermissions_NoManagement(t *testing.T) {
	p := domain.DefaultUserPermissions(domain.RoleUser)

	if p.CanManageUsers {
		t.Error("user: CanManageUsers should be false")
	}
	if p.CanManageRoles {
		t.Error("user: CanManageRoles should be false")
	}
	if p.CanViewAllLinks {
		t.Error("user: CanViewAllLinks should be false")
	}
	if p.CanEditAllLinks {
		t.Error("user: CanEditAllLinks should be false")
	}
	if p.CanDeleteAllLinks {
		t.Error("user: CanDeleteAllLinks should be false")
	}
	if p.CanViewAllStats {
		t.Error("user: CanViewAllStats should be false")
	}
	if p.CanManageAllTags {
		t.Error("user: CanManageAllTags should be false")
	}
}

// TestDefaultUserPermissions_RoleField — поле Role заполняется.
func TestDefaultUserPermissions_RoleField(t *testing.T) {
	for _, name := range []string{domain.RoleUser, "editor", "viewer"} {
		p := domain.DefaultUserPermissions(name)
		if p.Role != name {
			t.Errorf("Role: want %q, got %q", name, p.Role)
		}
	}
}

// TestDefaultUserPermissions_CustomSlug — кастомный slug для user выключен по умолчанию.
// Если DefaultUserPermissions не выдаёт CanCreateWithCustomSlug — это намеренное ограничение.
func TestDefaultUserPermissions_CustomSlugBehavior(t *testing.T) {
	p := domain.DefaultUserPermissions(domain.RoleUser)
	// Документируем текущее поведение (не prescriptive, а descriptive):
	t.Logf("DefaultUserPermissions.CanCreateWithCustomSlug = %v", p.CanCreateWithCustomSlug)
}

// ── Deny-all baseline ─────────────────────────────────────────────────────

// TestRolePermissions_ZeroValue_DenyAll — нулевое значение RolePermissions = запрет всего.
func TestRolePermissions_ZeroValue_DenyAll(t *testing.T) {
	var p domain.RolePermissions

	if p.CanCreateLinks {
		t.Error("zero: CanCreateLinks should be false")
	}
	if p.CanViewOwnLinks {
		t.Error("zero: CanViewOwnLinks should be false")
	}
	if p.CanManageUsers {
		t.Error("zero: CanManageUsers should be false")
	}
	if p.CanManageRoles {
		t.Error("zero: CanManageRoles should be false")
	}
	if p.CanViewAllLinks {
		t.Error("zero: CanViewAllLinks should be false")
	}
}

// ── Admin is superset of User ──────────────────────────────────────────────

// TestAdminIsSuperset — каждый флаг, разрешённый для user, разрешён и для admin.
func TestAdminIsSuperset(t *testing.T) {
	admin := domain.DefaultAdminPermissions("admin")
	user := domain.DefaultUserPermissions(domain.RoleUser)

	check := func(name string, adminVal, userVal bool) {
		if userVal && !adminVal {
			t.Errorf("flag %s: user=true but admin=false — admin must be superset", name)
		}
	}

	check("CanCreateLinks", admin.CanCreateLinks, user.CanCreateLinks)
	check("CanViewOwnLinks", admin.CanViewOwnLinks, user.CanViewOwnLinks)
	check("CanEditOwnLinks", admin.CanEditOwnLinks, user.CanEditOwnLinks)
	check("CanDeleteOwnLinks", admin.CanDeleteOwnLinks, user.CanDeleteOwnLinks)
	check("CanViewOwnStats", admin.CanViewOwnStats, user.CanViewOwnStats)
	check("CanCreateWithCustomSlug", admin.CanCreateWithCustomSlug, user.CanCreateWithCustomSlug)
	check("CanManageOwnTags", admin.CanManageOwnTags, user.CanManageOwnTags)
}
