package domain

import (
	"time"

	"github.com/google/uuid"
)

// Role — открытый тип: любая строка, задаваемая через ROLE_GROUPS в конфиге.
// RoleAdmin — единственная зарезервированная константа: используется только
// внутри RBAC-проверок (AdminOnly/RequireRole). Фактическое значение
// определяется ADMIN_ROLE в конфиге (config.AdminRole), по умолчанию "admin".
type Role = string

const (
	// RoleAdmin — значение по умолчанию для админской роли.
	// Используйте cfg.AdminRole там, где значение подставляется динамически.
	RoleAdmin Role = "admin"
	// RoleUser — оставлен для обратной совместимости. Не используйте в новом коде.
	RoleUser Role = "user"
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
	ShlinkAPIKey string    `db:"shlink_api_key"` // только внутри backend
	SlugPrefix   string    `db:"slug_prefix"`
	Status       Status    `db:"status"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

// Permissions вычисляются из роли пользователя. adminRole передаётся явно
// (берётся из config.AdminRole), чтобы не зависеть от глобального состояния.
type Permissions struct {
	CanCreateShortURL bool `json:"canCreateShortUrl"`
	CanEditOwnLinks   bool `json:"canEditOwnLinks"`
	CanDeleteOwnLinks bool `json:"canDeleteOwnLinks"`
	CanManageOwnTags  bool `json:"canManageOwnTags"`
	CanViewAuditLogs  bool `json:"canViewAuditLogs"`
	CanManageUsers    bool `json:"canManageUsers"`
}

func (u *User) ComputePermissions(adminRole string) Permissions {
	isAdmin := u.Role == adminRole
	return Permissions{
		CanCreateShortURL: true,
		CanEditOwnLinks:   true,
		CanDeleteOwnLinks: true,
		CanManageOwnTags:  true,
		CanViewAuditLogs:  isAdmin,
		CanManageUsers:    isAdmin,
	}
}
