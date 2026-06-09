package domain

import (
	"time"

	"github.com/google/uuid"
)

// Role — открытый тип: любая строка.
// Константы оставлены для совместимости с кодом, который ещё ссылается на них.
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

// Permission — атомарное действие, назначаемое роли.
type Permission struct {
	ID          uuid.UUID `db:"id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
}

// RoleEntity — роль пользователя.
type RoleEntity struct {
	ID          uuid.UUID    `db:"id"`
	Name        string       `db:"name"`
	Description string       `db:"description"`
	Permissions []Permission `db:"-"`
}

// User — доменная модель. ShlinkAPIKey НИКОГДА не сериализуется в HTTP-ответы.
// Role (текстовое поле users.role) сохраняется для совместимости c keycloak-провизионингом
// и для fallback-пути в PermissionService (когда role_id IS NULL).
type User struct {
	ID           uuid.UUID  `db:"id"`
	Sub          string     `db:"sub"`
	Username     string     `db:"username"`
	Email        string     `db:"email"`
	Role         string     `db:"role"`
	RoleID       *uuid.UUID `db:"role_id"`
	ShlinkAPIKey string     `db:"shlink_api_key"`
	SlugPrefix   string     `db:"slug_prefix"`
	Status       Status     `db:"status"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
}

// Константы имён разрешений — используются в хендлерах и сервисах при вызове permCtrl.Check.
const (
	PermDashboardView    = "dashboard.view"
	PermShortURLsCreate  = "short_urls.create"
	PermShortURLsUpdate  = "short_urls.update"
	PermShortURLsDelete  = "short_urls.delete"
	PermShortURLsViewAll = "short_urls.view_all"
	PermUsersView        = "users.view"
	PermUsersManage      = "users.manage"
	PermRolesView        = "roles.view"
	PermRolesManage      = "roles.manage"
	PermSystemConfig     = "system.config"
)
