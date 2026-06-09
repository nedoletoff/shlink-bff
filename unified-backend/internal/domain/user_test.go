package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"unified-backend/internal/domain"
)

func TestUser_Fields(t *testing.T) {
	id := uuid.New()
	roleID := uuid.New()
	u := domain.User{
		ID:       id,
		Sub:      "sub-123",
		Username: "nikita",
		Email:    "nikita@example.com",
		Role:     domain.RoleAdmin,
		RoleID:   &roleID,
		Status:   domain.StatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if u.ID != id {
		t.Errorf("ID mismatch")
	}
	if u.RoleID == nil || *u.RoleID != roleID {
		t.Errorf("RoleID mismatch")
	}
	if u.Role != "admin" {
		t.Errorf("Role: want admin, got %s", u.Role)
	}
	if u.ShlinkAPIKey != "" {
		t.Errorf("ShlinkAPIKey should be empty by default")
	}
}

func TestPermission_Fields(t *testing.T) {
	p := domain.Permission{
		ID:          uuid.New(),
		Name:        domain.PermShortURLsCreate,
		Description: "Create short URLs",
	}
	if p.Name != "short_urls.create" {
		t.Errorf("Permission name mismatch: %s", p.Name)
	}
}

func TestRoleEntity_Fields(t *testing.T) {
	r := domain.RoleEntity{
		ID:   uuid.New(),
		Name: "auditor_admin",
		Permissions: []domain.Permission{
			{ID: uuid.New(), Name: domain.PermUsersView},
			{ID: uuid.New(), Name: domain.PermRolesView},
		},
	}
	if len(r.Permissions) != 2 {
		t.Errorf("want 2 permissions, got %d", len(r.Permissions))
	}
}

func TestPermConstants(t *testing.T) {
	consts := []string{
		domain.PermDashboardView,
		domain.PermShortURLsCreate,
		domain.PermShortURLsUpdate,
		domain.PermShortURLsDelete,
		domain.PermShortURLsViewAll,
		domain.PermUsersView,
		domain.PermUsersManage,
		domain.PermRolesView,
		domain.PermRolesManage,
		domain.PermSystemConfig,
	}
	if len(consts) != 10 {
		t.Errorf("want 10 perm constants, got %d", len(consts))
	}
	for _, c := range consts {
		if c == "" {
			t.Errorf("empty perm constant")
		}
	}
}

func TestUser_RoleID_Nil(t *testing.T) {
	u := domain.User{
		ID:     uuid.New(),
		Role:   domain.RoleUser,
		RoleID: nil,
		Status: domain.StatusPending,
	}
	if u.RoleID != nil {
		t.Errorf("RoleID should be nil for unprovitioned user")
	}
}
