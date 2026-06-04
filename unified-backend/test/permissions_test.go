package test

import (
	"testing"

	"unified-backend/internal/domain"
)

// --- DefaultAdminPermissions ---

func TestDefaultAdminPermissions_AllGranted(t *testing.T) {
	p := domain.DefaultAdminPermissions("admin")

	if p.Role != "admin" {
		t.Errorf("Role: got %q", p.Role)
	}

	flags := map[string]bool{
		"CanViewOwnLinks":         p.CanViewOwnLinks,
		"CanViewAllLinks":         p.CanViewAllLinks,
		"CanCreateLinks":          p.CanCreateLinks,
		"CanCreateWithCustomSlug": p.CanCreateWithCustomSlug,
		"CanCreateWithoutSlug":    p.CanCreateWithoutSlug,
		"CanEditOwnLinks":         p.CanEditOwnLinks,
		"CanEditAllLinks":         p.CanEditAllLinks,
		"CanDeleteOwnLinks":       p.CanDeleteOwnLinks,
		"CanDeleteAllLinks":       p.CanDeleteAllLinks,
		"CanManageOwnTags":        p.CanManageOwnTags,
		"CanManageAllTags":        p.CanManageAllTags,
		"CanViewOwnStats":         p.CanViewOwnStats,
		"CanViewAllStats":         p.CanViewAllStats,
		"CanViewAuditLogs":        p.CanViewAuditLogs,
		"CanManageUsers":          p.CanManageUsers,
		"CanManageRoles":          p.CanManageRoles,
	}
	for name, val := range flags {
		if !val {
			t.Errorf("admin: %s should be true", name)
		}
	}
}

func TestDefaultAdminPermissions_CustomRoleName(t *testing.T) {
	p := domain.DefaultAdminPermissions("superadmin")
	if p.Role != "superadmin" {
		t.Errorf("expected role %q, got %q", "superadmin", p.Role)
	}
	if !p.CanManageRoles {
		t.Error("custom admin role should have CanManageRoles")
	}
}

// --- DefaultUserPermissions ---

func TestDefaultUserPermissions_OwnGranted(t *testing.T) {
	p := domain.DefaultUserPermissions("user")

	if p.Role != "user" {
		t.Errorf("Role: got %q", p.Role)
	}

	// CanCreateWithCustomSlug=true по умолчанию: пользователь имеет право на кастомный slug
	// на уровне permission. Реальное ограничение — feature-флаг UserCustomSlugEnabled
	// и обязательный slug_prefix (если UserSlugPrefixEnabled=true), проверяемые в
	// ShlinkService.EnforceSlugPrefix.
	grantedFlags := map[string]bool{
		"CanViewOwnLinks":         p.CanViewOwnLinks,
		"CanCreateLinks":          p.CanCreateLinks,
		"CanCreateWithCustomSlug": p.CanCreateWithCustomSlug,
		"CanCreateWithoutSlug":    p.CanCreateWithoutSlug,
		"CanEditOwnLinks":         p.CanEditOwnLinks,
		"CanDeleteOwnLinks":       p.CanDeleteOwnLinks,
		"CanManageOwnTags":        p.CanManageOwnTags,
		"CanViewOwnStats":         p.CanViewOwnStats,
	}
	for name, val := range grantedFlags {
		if !val {
			t.Errorf("user: %s should be true", name)
		}
	}
}

func TestDefaultUserPermissions_AllDenied(t *testing.T) {
	p := domain.DefaultUserPermissions("user")

	// CanCreateWithCustomSlug намеренно исключён: permission теперь выдаётся по умолчанию,
	// доступность кастомного slug контролируется feature-флагом UserCustomSlugEnabled
	// (см. TestEnforceSlugPrefix_UserCustomSlugFeatureDisabled в service_test.go).
	deniedFlags := map[string]bool{
		"CanViewAllLinks":   p.CanViewAllLinks,
		"CanEditAllLinks":   p.CanEditAllLinks,
		"CanDeleteAllLinks": p.CanDeleteAllLinks,
		"CanManageAllTags":  p.CanManageAllTags,
		"CanViewAllStats":   p.CanViewAllStats,
		"CanViewAuditLogs":  p.CanViewAuditLogs,
		"CanManageUsers":    p.CanManageUsers,
		"CanManageRoles":    p.CanManageRoles,
	}
	for name, val := range deniedFlags {
		if val {
			t.Errorf("user: %s should be false", name)
		}
	}
}

func TestDefaultUserPermissions_CustomRoleName(t *testing.T) {
	p := domain.DefaultUserPermissions("editor")
	if p.Role != "editor" {
		t.Errorf("expected role %q, got %q", "editor", p.Role)
	}
	// editor наследует базовые user-права
	if !p.CanCreateLinks {
		t.Error("custom user-tier role should have CanCreateLinks")
	}
	if p.CanManageUsers {
		t.Error("custom user-tier role should not have CanManageUsers")
	}
}

// --- Разграничение admin vs user ---

func TestPermissions_AdminHasAllUserCan(t *testing.T) {
	admin := domain.DefaultAdminPermissions("admin")
	user := domain.DefaultUserPermissions("user")

	// Всё что может user — должен уметь и admin.
	type check struct {
		name     string
		userCan  bool
		adminCan bool
	}
	checks := []check{
		{"CanViewOwnLinks", user.CanViewOwnLinks, admin.CanViewOwnLinks},
		{"CanCreateLinks", user.CanCreateLinks, admin.CanCreateLinks},
		{"CanCreateWithCustomSlug", user.CanCreateWithCustomSlug, admin.CanCreateWithCustomSlug},
		{"CanEditOwnLinks", user.CanEditOwnLinks, admin.CanEditOwnLinks},
		{"CanDeleteOwnLinks", user.CanDeleteOwnLinks, admin.CanDeleteOwnLinks},
		{"CanManageOwnTags", user.CanManageOwnTags, admin.CanManageOwnTags},
		{"CanViewOwnStats", user.CanViewOwnStats, admin.CanViewOwnStats},
	}
	for _, c := range checks {
		if c.userCan && !c.adminCan {
			t.Errorf("%s: user has permission but admin does not — invariant violated", c.name)
		}
	}
}

func TestPermissions_OnlyAdminHasElevated(t *testing.T) {
	admin := domain.DefaultAdminPermissions("admin")
	user := domain.DefaultUserPermissions("user")

	// Эти флаги — только для admin.
	// CanCreateWithCustomSlug намеренно исключён из этого списка:
	// он выдаётся и пользователям, ограничение переехало на уровень
	// feature-флага UserCustomSlugEnabled (см. service/shlink_service.go).
	elevated := map[string]struct{ a, u bool }{
		"CanViewAllLinks":   {admin.CanViewAllLinks, user.CanViewAllLinks},
		"CanEditAllLinks":   {admin.CanEditAllLinks, user.CanEditAllLinks},
		"CanDeleteAllLinks": {admin.CanDeleteAllLinks, user.CanDeleteAllLinks},
		"CanManageAllTags":  {admin.CanManageAllTags, user.CanManageAllTags},
		"CanViewAllStats":   {admin.CanViewAllStats, user.CanViewAllStats},
		"CanViewAuditLogs":  {admin.CanViewAuditLogs, user.CanViewAuditLogs},
		"CanManageUsers":    {admin.CanManageUsers, user.CanManageUsers},
		"CanManageRoles":    {admin.CanManageRoles, user.CanManageRoles},
	}
	for name, v := range elevated {
		if !v.a {
			t.Errorf("%s: admin should have this permission", name)
		}
		if v.u {
			t.Errorf("%s: user should NOT have this permission", name)
		}
	}
}

// --- RolePermissions zero-value ---

// Нулевое значение RolePermissions — полный запрет.
// Это важно: если permissions не загружены из БД, нет "тихого" расширения прав.
func TestRolePermissions_ZeroValueIsDenyAll(t *testing.T) {
	var p domain.RolePermissions

	flags := map[string]bool{
		"CanViewOwnLinks":         p.CanViewOwnLinks,
		"CanViewAllLinks":         p.CanViewAllLinks,
		"CanCreateLinks":          p.CanCreateLinks,
		"CanCreateWithCustomSlug": p.CanCreateWithCustomSlug,
		"CanCreateWithoutSlug":    p.CanCreateWithoutSlug,
		"CanEditOwnLinks":         p.CanEditOwnLinks,
		"CanEditAllLinks":         p.CanEditAllLinks,
		"CanDeleteOwnLinks":       p.CanDeleteOwnLinks,
		"CanDeleteAllLinks":       p.CanDeleteAllLinks,
		"CanManageOwnTags":        p.CanManageOwnTags,
		"CanManageAllTags":        p.CanManageAllTags,
		"CanViewOwnStats":         p.CanViewOwnStats,
		"CanViewAllStats":         p.CanViewAllStats,
		"CanViewAuditLogs":        p.CanViewAuditLogs,
		"CanManageUsers":          p.CanManageUsers,
		"CanManageRoles":          p.CanManageRoles,
	}
	for name, val := range flags {
		if val {
			t.Errorf("zero RolePermissions: %s should be false (deny-all)", name)
		}
	}
}
