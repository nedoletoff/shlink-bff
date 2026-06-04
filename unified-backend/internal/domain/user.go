package domain

import (
	"time"

	"github.com/google/uuid"
)

// Role — открытый тип: любая строка, задаваемая через ROLE_GROUPS в конфиге.
type Role = string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

type Status string

const (
	StatusActive   Status = "active"
	StatusDisabled Status = "disabled"
	StatusPending  Status = "pending"
)

// User — доменная модель. ShlinkAPIKey НИКОГДА не сериализуется в HTTP-ответы.
type User struct {
	ID           uuid.UUID `db:"id"`
	Sub          string    `db:"sub"`
	Username     string    `db:"username"`
	Email        string    `db:"email"`
	Role         Role      `db:"role"`
	ShlinkAPIKey string    `db:"shlink_api_key"`
	SlugPrefix   string    `db:"slug_prefix"`
	Status       Status    `db:"status"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

// RolePermissions — набор флагов для роли, хранится в таблице role_permissions.
// Загружается при старте и кешируется в PermissionsCache.
type RolePermissions struct {
	Role string `db:"role" json:"role"`

	// Просмотр ссылок
	CanViewOwnLinks bool `db:"can_view_own_links" json:"canViewOwnLinks"`
	CanViewAllLinks bool `db:"can_view_all_links" json:"canViewAllLinks"`

	// Создание ссылок
	CanCreateLinks           bool `db:"can_create_links"             json:"canCreateLinks"`
	CanCreateWithCustomSlug  bool `db:"can_create_with_custom_slug"  json:"canCreateWithCustomSlug"`
	CanCreateWithoutSlug     bool `db:"can_create_without_slug"      json:"canCreateWithoutSlug"`

	// Редактирование
	CanEditOwnLinks bool `db:"can_edit_own_links" json:"canEditOwnLinks"`
	CanEditAllLinks bool `db:"can_edit_all_links" json:"canEditAllLinks"`

	// Удаление
	CanDeleteOwnLinks bool `db:"can_delete_own_links" json:"canDeleteOwnLinks"`
	CanDeleteAllLinks bool `db:"can_delete_all_links" json:"canDeleteAllLinks"`

	// Теги
	CanManageOwnTags bool `db:"can_manage_own_tags" json:"canManageOwnTags"`
	CanManageAllTags bool `db:"can_manage_all_tags" json:"canManageAllTags"`

	// Статистика
	CanViewOwnStats bool `db:"can_view_own_stats" json:"canViewOwnStats"`
	CanViewAllStats bool `db:"can_view_all_stats" json:"canViewAllStats"`

	// Управление
	CanViewAuditLogs bool `db:"can_view_audit_logs" json:"canViewAuditLogs"`
	CanManageUsers   bool `db:"can_manage_users"   json:"canManageUsers"`
	CanManageRoles   bool `db:"can_manage_roles"   json:"canManageRoles"`

	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

// DefaultAdminPermissions — полные права для fallback если БД недоступна.
func DefaultAdminPermissions(role string) RolePermissions {
	return RolePermissions{
		Role:                    role,
		CanViewOwnLinks:         true,
		CanViewAllLinks:         true,
		CanCreateLinks:          true,
		CanCreateWithCustomSlug: true,
		CanCreateWithoutSlug:    true,
		CanEditOwnLinks:         true,
		CanEditAllLinks:         true,
		CanDeleteOwnLinks:       true,
		CanDeleteAllLinks:       true,
		CanManageOwnTags:        true,
		CanManageAllTags:        true,
		CanViewOwnStats:         true,
		CanViewAllStats:         true,
		CanViewAuditLogs:        true,
		CanManageUsers:          true,
		CanManageRoles:          true,
	}
}

// DefaultUserPermissions — минимальные права для fallback.
func DefaultUserPermissions(role string) RolePermissions {
	return RolePermissions{
		Role:                    role,
		CanViewOwnLinks:         true,
		CanCreateLinks:          true,
		CanCreateWithCustomSlug: true,
		CanCreateWithoutSlug:    true,
		CanEditOwnLinks:         true,
		CanDeleteOwnLinks:       true,
		CanManageOwnTags:        true,
		CanViewOwnStats:         true,
	}
}
