package test

import (
	"testing"

	"unified-backend/internal/domain"
)

// ─── IsAdminRole ─────────────────────────────────────────────────────────────

func TestIsAdminRole_Admin(t *testing.T) {
	if !domain.IsAdminRole(domain.RoleAdmin) {
		t.Error("RoleAdmin must be admin")
	}
}

func TestIsAdminRole_UserString(t *testing.T) {
	if domain.IsAdminRole(domain.Role("user")) {
		t.Error("role=user must NOT be admin")
	}
}

func TestIsAdminRole_CustomRole(t *testing.T) {
	for _, r := range []string{"viewer", "editor", "moderator", "readonly", ""} {
		if domain.IsAdminRole(domain.Role(r)) {
			t.Errorf("custom role %q must NOT be admin", r)
		}
	}
}

// ─── ComputePermissions: admin ────────────────────────────────────────────────

func TestComputePermissions_Admin(t *testing.T) {
	user := &domain.User{Role: domain.RoleAdmin}
	perms := user.ComputePermissions()

	if !perms.CanViewAuditLogs {
		t.Error("admin should CanViewAuditLogs")
	}
	if !perms.CanManageUsers {
		t.Error("admin should CanManageUsers")
	}
	if !perms.CanCreateShortURL {
		t.Error("admin should CanCreateShortURL")
	}
	if !perms.CanEditOwnLinks {
		t.Error("admin should CanEditOwnLinks")
	}
	if !perms.CanDeleteOwnLinks {
		t.Error("admin should CanDeleteOwnLinks")
	}
	if !perms.CanManageOwnTags {
		t.Error("admin should CanManageOwnTags")
	}
}

// ─── ComputePermissions: произвольные user-роли ───────────────────────────────

func TestComputePermissions_UserRoleString(t *testing.T) {
	user := &domain.User{Role: domain.Role("user")}
	perms := user.ComputePermissions()

	if perms.CanViewAuditLogs {
		t.Error("role=user must NOT CanViewAuditLogs")
	}
	if perms.CanManageUsers {
		t.Error("role=user must NOT CanManageUsers")
	}
	if !perms.CanCreateShortURL {
		t.Error("role=user SHOULD CanCreateShortURL")
	}
}

func TestComputePermissions_CustomRoleViewer(t *testing.T) {
	viewer := &domain.User{Role: domain.Role("viewer")}
	perms := viewer.ComputePermissions()

	// default user permissions apply
	if perms.CanManageUsers {
		t.Error("viewer must NOT CanManageUsers")
	}
	if perms.CanViewAuditLogs {
		t.Error("viewer must NOT CanViewAuditLogs")
	}
	if !perms.CanCreateShortURL {
		t.Error("viewer SHOULD CanCreateShortURL (default)")
	}
}

func TestComputePermissions_CustomRoleEditor(t *testing.T) {
	editor := &domain.User{Role: domain.Role("editor")}
	perms := editor.ComputePermissions()

	if perms.CanManageUsers {
		t.Error("editor must NOT CanManageUsers")
	}
	if !perms.CanEditOwnLinks {
		t.Error("editor SHOULD CanEditOwnLinks (default)")
	}
}

// ─── ComputePermissions: RolePermissionsOverride ──────────────────────────────

func TestComputePermissions_Override_RestrictedViewer(t *testing.T) {
	// Регистрируем кастомные права для viewer
	domain.RolePermissionsOverride[domain.Role("viewer-restricted")] = domain.Permissions{
		CanCreateShortURL: false,
		CanEditOwnLinks:   false,
		CanDeleteOwnLinks: false,
		CanManageOwnTags:  false,
		CanViewAuditLogs:  false,
		CanManageUsers:    false,
	}
	defer delete(domain.RolePermissionsOverride, domain.Role("viewer-restricted"))

	u := &domain.User{Role: domain.Role("viewer-restricted")}
	perms := u.ComputePermissions()

	if perms.CanCreateShortURL {
		t.Error("viewer-restricted must NOT CanCreateShortURL")
	}
	if perms.CanEditOwnLinks {
		t.Error("viewer-restricted must NOT CanEditOwnLinks")
	}
}

func TestComputePermissions_Override_PowerUser(t *testing.T) {
	// power-user: всё кроме canManageUsers
	domain.RolePermissionsOverride[domain.Role("power-user")] = domain.Permissions{
		CanCreateShortURL: true,
		CanEditOwnLinks:   true,
		CanDeleteOwnLinks: true,
		CanManageOwnTags:  true,
		CanViewAuditLogs:  true,
		CanManageUsers:    false,
	}
	defer delete(domain.RolePermissionsOverride, domain.Role("power-user"))

	u := &domain.User{Role: domain.Role("power-user")}
	perms := u.ComputePermissions()

	if !perms.CanViewAuditLogs {
		t.Error("power-user SHOULD CanViewAuditLogs")
	}
	if perms.CanManageUsers {
		t.Error("power-user must NOT CanManageUsers")
	}
}

func TestComputePermissions_Override_DoesNotAffectAdmin(t *testing.T) {
	// Попытка переопределить admin через override — должна работать,
	// потому что override имеет наивысший приоритет
	domain.RolePermissionsOverride[domain.RoleAdmin] = domain.Permissions{
		CanCreateShortURL: false,
		CanManageUsers:    false,
	}
	defer delete(domain.RolePermissionsOverride, domain.RoleAdmin)

	u := &domain.User{Role: domain.RoleAdmin}
	perms := u.ComputePermissions()

	// Override явно выставил false — ожидаем false
	if perms.CanCreateShortURL {
		t.Error("overridden admin: CanCreateShortURL should be false per override")
	}
}

func TestComputePermissions_Override_NotRegistered_FallsBackToDefault(t *testing.T) {
	// Убеждаемся что роль без override получает DefaultUserPermissions
	u := &domain.User{Role: domain.Role("unknown-role-xyz")}
	perms := u.ComputePermissions()

	if perms != domain.DefaultUserPermissions {
		t.Errorf("unknown role should get DefaultUserPermissions, got %+v", perms)
	}
}

// ─── DefaultUserPermissions: изменение через var ──────────────────────────────

func TestDefaultUserPermissions_Override(t *testing.T) {
	orig := domain.DefaultUserPermissions
	defer func() { domain.DefaultUserPermissions = orig }()

	domain.DefaultUserPermissions = domain.Permissions{
		CanCreateShortURL: false,
		CanEditOwnLinks:   false,
		CanDeleteOwnLinks: false,
		CanManageOwnTags:  false,
		CanViewAuditLogs:  false,
		CanManageUsers:    false,
	}

	u := &domain.User{Role: domain.Role("any")}
	perms := u.ComputePermissions()
	if perms.CanCreateShortURL {
		t.Error("after override DefaultUserPermissions, CanCreateShortURL should be false")
	}
}
