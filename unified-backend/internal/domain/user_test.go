package domain_test

import (
	"testing"

	"unified-backend/internal/domain"
)

func TestDefaultAdminPermissions_AllTrue(t *testing.T) {
	p := domain.DefaultAdminPermissions("admin")
	if p.Role != "admin" {
		t.Errorf("role: want admin, got %s", p.Role)
	}
	flags := []struct {
		name string
		v    bool
	}{
		{"CanViewOwnLinks", p.CanViewOwnLinks},
		{"CanViewAllLinks", p.CanViewAllLinks},
		{"CanCreateLinks", p.CanCreateLinks},
		{"CanCreateWithCustomSlug", p.CanCreateWithCustomSlug},
		{"CanCreateWithoutSlug", p.CanCreateWithoutSlug},
		{"CanEditOwnLinks", p.CanEditOwnLinks},
		{"CanEditAllLinks", p.CanEditAllLinks},
		{"CanDeleteOwnLinks", p.CanDeleteOwnLinks},
		{"CanDeleteAllLinks", p.CanDeleteAllLinks},
		{"CanManageOwnTags", p.CanManageOwnTags},
		{"CanManageAllTags", p.CanManageAllTags},
		{"CanViewOwnStats", p.CanViewOwnStats},
		{"CanViewAllStats", p.CanViewAllStats},
		{"CanViewAuditLogs", p.CanViewAuditLogs},
		{"CanManageUsers", p.CanManageUsers},
		{"CanManageRoles", p.CanManageRoles},
		{"CanManageSettings", p.CanManageSettings},
	}
	for _, f := range flags {
		if !f.v {
			t.Errorf("admin: %s should be true", f.name)
		}
	}
}

func TestDefaultUserPermissions_MinimalSet(t *testing.T) {
	p := domain.DefaultUserPermissions("user")
	if p.Role != "user" {
		t.Errorf("role: want user, got %s", p.Role)
	}
	// должны быть true
	mustTrue := []struct {
		name string
		v    bool
	}{
		{"CanViewOwnLinks", p.CanViewOwnLinks},
		{"CanCreateLinks", p.CanCreateLinks},
		{"CanCreateWithCustomSlug", p.CanCreateWithCustomSlug},
		{"CanCreateWithoutSlug", p.CanCreateWithoutSlug},
		{"CanEditOwnLinks", p.CanEditOwnLinks},
		{"CanDeleteOwnLinks", p.CanDeleteOwnLinks},
		{"CanManageOwnTags", p.CanManageOwnTags},
		{"CanViewOwnStats", p.CanViewOwnStats},
	}
	for _, f := range mustTrue {
		if !f.v {
			t.Errorf("user: %s should be true", f.name)
		}
	}
	// должны быть false
	mustFalse := []struct {
		name string
		v    bool
	}{
		{"CanViewAllLinks", p.CanViewAllLinks},
		{"CanEditAllLinks", p.CanEditAllLinks},
		{"CanDeleteAllLinks", p.CanDeleteAllLinks},
		{"CanManageAllTags", p.CanManageAllTags},
		{"CanViewAllStats", p.CanViewAllStats},
		{"CanViewAuditLogs", p.CanViewAuditLogs},
		{"CanManageUsers", p.CanManageUsers},
		{"CanManageRoles", p.CanManageRoles},
		{"CanManageSettings", p.CanManageSettings},
	}
	for _, f := range mustFalse {
		if f.v {
			t.Errorf("user: %s should be false", f.name)
		}
	}
}
