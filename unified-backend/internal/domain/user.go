package domain

import (
	"time"

	"github.com/google/uuid"
)

// Role — открытый тип: любая строка.
// Константы оставлены для совместимости с кодом, который ещё ссылается на них.
type Role = string

const (
	RoleAdmin    Role = "admin"
	RoleUser     Role = "user"
	RoleStandard Role = "standard" // новая: роль по умолчанию (DEFAULT_ROLE)
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
	ID             uuid.UUID  `db:"id"`
	Sub            string     `db:"sub"`
	Username       string     `db:"username"`
	Email          string     `db:"email"`
	Role           string     `db:"role"`
	RoleID         *uuid.UUID `db:"role_id"`
	ShlinkAPIKey   string     `db:"shlink_api_key"`
	SlugPrefix     string     `db:"slug_prefix"`
	// AllowedDomains — JSON-массив разрешённых доменов (например '["example.com","short.io"]').
	// Пустая строка или '[]' означает — ограничений нет.
	AllowedDomains string     `db:"allowed_domains"`
	Status         Status     `db:"status"`
	CreatedAt      time.Time  `db:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at"`
}

// Константы имён разрешений — используются в хендлерах и сервисах при вызове permCtrl.Check.
const (
	PermDashboardView    = "dashboard.view"
	PermShortURLsCreate  = "short_urls.create"      // создать любую ссылку (admin/manager)
	PermShortURLsUpdate  = "short_urls.update"
	PermShortURLsDelete  = "short_urls.delete"      // удалить чужую ссылку (global)
	PermShortURLsViewAll = "short_urls.view_all"
	PermUsersView        = "users.view"
	PermUsersManage      = "users.manage"
	PermRolesView        = "roles.view"
	PermRolesManage      = "roles.manage"
	PermSystemConfig     = "system.config"

	// Права роли standard — действия с владельческими объектами
	PermShortURLsCreateOwn = "short_urls.create.own" // создать ссылку (владелец)
	PermShortURLsDeleteOwn = "short_urls.delete.own" // удалить свою ссылку
	PermShortURLsViewOwn   = "short_urls.view.own"   // просматривать свои ссылки
	PermShortURLsStatsOwn  = "short_urls.stats.own"  // статистика своих ссылок

	// Ограничения по домену и префиксу (информационные константы; хранятся в users, не в permissions)
	PermShortURLsRestrictDomain = "short_urls.restrict.domain" // ограничение по разрешённым доменам
	PermShortURLsRestrictPrefix = "short_urls.restrict.prefix" // ограничение по slug-префиксу
)

// AllPermissions — полный список всех системных разрешений.
// Используется GET /api/permissions.
var AllPermissions = []string{
	PermDashboardView,
	PermShortURLsCreate,
	PermShortURLsUpdate,
	PermShortURLsDelete,
	PermShortURLsViewAll,
	PermShortURLsCreateOwn,
	PermShortURLsDeleteOwn,
	PermShortURLsViewOwn,
	PermShortURLsStatsOwn,
	PermUsersView,
	PermUsersManage,
	PermRolesView,
	PermRolesManage,
	PermSystemConfig,
}
