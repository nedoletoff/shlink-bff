package domain

import (
	"time"

	"github.com/google/uuid"
)

type Role = string

const (
	RoleAdmin    Role = "admin"
	RoleUser     Role = "user"
	RoleStandard Role = "standard"
)

type Status string

const (
	StatusActive   Status = "active"
	StatusDisabled Status = "disabled"
	StatusPending  Status = "pending"
)

// Permission – атомарное действие
type Permission struct {
	ID          uuid.UUID `db:"id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
}

// RoleEntity – роль с набором разрешений
type RoleEntity struct {
	ID          uuid.UUID    `db:"id"`
	Name        string       `db:"name"`
	Description string       `db:"description"`
	Permissions []Permission `db:"-"`
}

// User – доменная модель
type User struct {
	ID             uuid.UUID  `db:"id"`
	Sub            string     `db:"sub"`
	Username       string     `db:"username"`
	Email          string     `db:"email"`
	Role           string     `db:"role_text"` // денормализованное имя роли
	RoleID         *uuid.UUID `db:"role_id"`
	ShlinkAPIKey   string     `db:"shlink_api_key"`
	SlugPrefix     string     `db:"slug_prefix"`
	AllowedDomains string     `db:"allowed_domains"` // JSON array
	Status         Status     `db:"status"`
	CreatedAt      time.Time  `db:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at"`
}

// ─── Разрешения (permissions) ─────────────────────────────────────────────

const (
	// Базовые
	PermDashboardView = "dashboard.view"
	PermSystemConfig  = "system.config" // полный доступ к конфигу (чтение+запись)

	// Просмотр конфигурации (только чтение)
	PermSystemConfigView = "system.config.view"

	// Ссылки – общие действия
	PermShortURLsCreate  = "short_urls.create"   // создание любой ссылки (без детализации own/all)
	PermShortURLsUpdate  = "short_urls.update"   // редактирование любой ссылки
	PermShortURLsDelete  = "short_urls.delete"   // удаление любой ссылки
	PermShortURLsViewAll = "short_urls.view_all" // просмотр чужих ссылок

	// Свои ссылки (own)
	PermShortURLsCreateOwn     = "short_urls.create.own"
	PermShortURLsUpdateOwn     = "short_urls.update.own"
	PermShortURLsDeleteOwn     = "short_urls.delete.own"
	PermShortURLsDeactivateOwn = "short_urls.deactivate.own"
	PermShortURLsReactivateOwn = "short_urls.reactivate.own"
	PermShortURLsViewStatsOwn  = "short_urls.view_stats.own"
	PermShortURLsManageTagsOwn = "short_urls.manage_tags.own"

	// Чужие ссылки (all)
	PermShortURLsUpdateAll     = "short_urls.update.all"
	PermShortURLsDeleteAll     = "short_urls.delete.all"
	PermShortURLsDeactivateAll = "short_urls.deactivate.all"
	PermShortURLsReactivateAll = "short_urls.reactivate.all"
	PermShortURLsViewStatsAll  = "short_urls.view_stats.all"
	PermShortURLsManageTagsAll = "short_urls.manage_tags.all"

	// Дополнительные возможности при создании/обновлении ссылки
	PermShortURLsCustomSlug  = "short_urls.custom_slug"  // разрешён кастомный slug
	PermShortURLsTimeLimits  = "short_urls.time_limits"  // разрешены validSince/validUntil
	PermShortURLsVisitLimits = "short_urls.visit_limits" // разрешён maxVisits

	// Пользователи и роли
	PermUsersView   = "users.view"
	PermUsersManage = "users.manage"
	PermRolesView   = "roles.view"
	PermRolesManage = "roles.manage"
)

// AllPermissions – полный список разрешений для GET /api/permissions
var AllPermissions = []string{
	PermDashboardView,
	PermSystemConfig,
	PermSystemConfigView,

	PermShortURLsCreate,
	PermShortURLsUpdate,
	PermShortURLsDelete,
	PermShortURLsViewAll,

	PermShortURLsCreateOwn,
	PermShortURLsUpdateOwn,
	PermShortURLsDeleteOwn,
	PermShortURLsDeactivateOwn,
	PermShortURLsReactivateOwn,
	PermShortURLsViewStatsOwn,
	PermShortURLsManageTagsOwn,

	PermShortURLsUpdateAll,
	PermShortURLsDeleteAll,
	PermShortURLsDeactivateAll,
	PermShortURLsReactivateAll,
	PermShortURLsViewStatsAll,
	PermShortURLsManageTagsAll,

	PermShortURLsCustomSlug,
	PermShortURLsTimeLimits,
	PermShortURLsVisitLimits,

	PermUsersView,
	PermUsersManage,
	PermRolesView,
	PermRolesManage,
}

