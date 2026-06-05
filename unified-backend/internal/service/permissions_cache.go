package service

import (
	"context"
	"log/slog"
	"sync"

	"unified-backend/internal/domain"
	"unified-backend/internal/repository/postgres"
)

// PermissionsCache — потокобезопасный in-memory кеш permissions по роли.
// Загружается при старте, обновляется при изменении через admin API.
// Если роль отсутствует в кеше — возвращаются нулевые права (deny by default).
type PermissionsCache struct {
	mu    sync.RWMutex
	cache map[string]domain.RolePermissions
	repo  *postgres.RolePermissionsRepository
	admin string // adminRole из cfg — получает DefaultAdminPermissions при отсутствии в БД
}

func NewPermissionsCache(repo *postgres.RolePermissionsRepository, adminRole string) *PermissionsCache {
	return &PermissionsCache{
		cache: make(map[string]domain.RolePermissions),
		repo:  repo,
		admin: adminRole,
	}
}

// Load загружает все permissions из БД в кеш. Вызывается при старте.
func (c *PermissionsCache) Load(ctx context.Context) error {
	all, err := c.repo.GetAll(ctx)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, p := range all {
		c.cache[p.Role] = p
	}
	slog.Info("permissions cache loaded", "roles", len(c.cache))
	return nil
}

// Get возвращает permissions для одной роли.
// Если роль не найдена и это adminRole — возвращает DefaultAdminPermissions.
// Иначе — DefaultUserPermissions (deny all management флаги).
func (c *PermissionsCache) Get(role string) domain.RolePermissions {
	c.mu.RLock()
	p, ok := c.cache[role]
	c.mu.RUnlock()
	if ok {
		return p
	}
	if role == c.admin {
		slog.Warn("permissions cache: admin role not in DB, using defaults", "role", role)
		return domain.DefaultAdminPermissions(role)
	}
	slog.Warn("permissions cache: unknown role, using minimal defaults", "role", role)
	return domain.DefaultUserPermissions(role)
}

// GetMerged возвращает объединённые permissions для набора ролей (OR-семантика).
// Если пользователь имеет несколько ролей — получает все флаги, разрешённые хотя бы в одной.
// Поле Role в результате — пустая строка (merged не привязан к одной роли).
func (c *PermissionsCache) GetMerged(roles []string) domain.RolePermissions {
	if len(roles) == 0 {
		return domain.RolePermissions{}
	}
	if len(roles) == 1 {
		return c.Get(roles[0])
	}
	merged := domain.RolePermissions{Role: ""}
	for _, r := range roles {
		p := c.Get(r)
		merged.CanViewOwnLinks = merged.CanViewOwnLinks || p.CanViewOwnLinks
		merged.CanViewAllLinks = merged.CanViewAllLinks || p.CanViewAllLinks
		merged.CanCreateLinks = merged.CanCreateLinks || p.CanCreateLinks
		merged.CanCreateWithCustomSlug = merged.CanCreateWithCustomSlug || p.CanCreateWithCustomSlug
		merged.CanCreateWithoutSlug = merged.CanCreateWithoutSlug || p.CanCreateWithoutSlug
		merged.CanEditOwnLinks = merged.CanEditOwnLinks || p.CanEditOwnLinks
		merged.CanEditAllLinks = merged.CanEditAllLinks || p.CanEditAllLinks
		merged.CanDeleteOwnLinks = merged.CanDeleteOwnLinks || p.CanDeleteOwnLinks
		merged.CanDeleteAllLinks = merged.CanDeleteAllLinks || p.CanDeleteAllLinks
		merged.CanManageOwnTags = merged.CanManageOwnTags || p.CanManageOwnTags
		merged.CanManageAllTags = merged.CanManageAllTags || p.CanManageAllTags
		merged.CanViewOwnStats = merged.CanViewOwnStats || p.CanViewOwnStats
		merged.CanViewAllStats = merged.CanViewAllStats || p.CanViewAllStats
		merged.CanViewAuditLogs = merged.CanViewAuditLogs || p.CanViewAuditLogs
		merged.CanManageUsers = merged.CanManageUsers || p.CanManageUsers
		merged.CanManageRoles = merged.CanManageRoles || p.CanManageRoles
	}
	return merged
}

// Set обновляет одну роль в кеше (вызывается после Upsert в БД).
func (c *PermissionsCache) Set(p domain.RolePermissions) {
	c.mu.Lock()
	c.cache[p.Role] = p
	c.mu.Unlock()
}

// GetAll возвращает снимок всего кеша (для /api/admin/roles).
func (c *PermissionsCache) GetAll() []domain.RolePermissions {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]domain.RolePermissions, 0, len(c.cache))
	for _, p := range c.cache {
		out = append(out, p)
	}
	return out
}
