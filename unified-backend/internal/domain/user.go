package domain

import (
	"time"

	"github.com/google/uuid"
)

type Role   string
type Status string

const (
	// RoleAdmin — единственная захардкоженная привилегированная роль.
	// Имя можно переопределить через adminRoles в middleware/identity.go.
	RoleAdmin Role = "admin"

	// RoleUser больше не существует как константа.
	// Любая роль, не входящая в набор admin-групп, считается непривилегированной.
	// Используйте произвольную строку: Role("viewer"), Role("editor"), etc.

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

// Permissions вычисляются из роли пользователя.
type Permissions struct {
	CanCreateShortURL bool `json:"canCreateShortUrl"`
	CanEditOwnLinks   bool `json:"canEditOwnLinks"`
	CanDeleteOwnLinks bool `json:"canDeleteOwnLinks"`
	CanManageOwnTags  bool `json:"canManageOwnTags"`
	CanViewAuditLogs  bool `json:"canViewAuditLogs"`
	CanManageUsers    bool `json:"canManageUsers"`
}

// IsAdminRole возвращает true, если роль является административной.
// На сегодня это только RoleAdmin, но функция позволяет расширить логику
// без изменения всех call-sites.
func IsAdminRole(r Role) bool {
	return r == RoleAdmin
}

// DefaultUserPermissions — права, которые получает любая непривилегированная роль.
// Вынесена в переменную, чтобы тесты могли её подменить без сайд-эффектов.
var DefaultUserPermissions = Permissions{
	CanCreateShortURL: true,
	CanEditOwnLinks:   true,
	CanDeleteOwnLinks: true,
	CanManageOwnTags:  true,
	CanViewAuditLogs:  false,
	CanManageUsers:    false,
}

// RolePermissionsOverride позволяет задать кастомные права для конкретной роли.
// Если роль не найдена в map — применяются DefaultUserPermissions (для user-ролей)
// или полные права (для admin).
// Пример:
//   domain.RolePermissionsOverride[Role("viewer")] = domain.Permissions{
//       CanCreateShortURL: false, CanEditOwnLinks: false, ...
//   }
var RolePermissionsOverride = map[Role]Permissions{}

// ComputePermissions вычисляет права пользователя с учётом:
// 1. Кастомных переопределений (RolePermissionsOverride)
// 2. Административной роли (IsAdminRole)
// 3. Дефолтных прав для непривилегированных ролей
func (u *User) ComputePermissions() Permissions {
	// Кастомное переопределение имеет наивысший приоритет
	if override, ok := RolePermissionsOverride[u.Role]; ok {
		return override
	}
	if IsAdminRole(u.Role) {
		return Permissions{
			CanCreateShortURL: true,
			CanEditOwnLinks:   true,
			CanDeleteOwnLinks: true,
			CanManageOwnTags:  true,
			CanViewAuditLogs:  true,
			CanManageUsers:    true,
		}
	}
	return DefaultUserPermissions
}
