package domain

import (
	"time"

	"github.com/google/uuid"
)

type Role   string
type Status string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"

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

// Permissions вычисляются из роли пользователя
type Permissions struct {
	CanCreateShortURL bool `json:"canCreateShortUrl"`
	CanEditOwnLinks   bool `json:"canEditOwnLinks"`
	CanDeleteOwnLinks bool `json:"canDeleteOwnLinks"`
	CanManageOwnTags  bool `json:"canManageOwnTags"`
	CanViewAuditLogs  bool `json:"canViewAuditLogs"`
	CanManageUsers    bool `json:"canManageUsers"`
}

func (u *User) ComputePermissions() Permissions {
	isAdmin := u.Role == RoleAdmin
	return Permissions{
		CanCreateShortURL: true,
		CanEditOwnLinks:   true,
		CanDeleteOwnLinks: true,
		CanManageOwnTags:  true,
		CanViewAuditLogs:  isAdmin,
		CanManageUsers:    isAdmin,
	}
}
