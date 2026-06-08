package service

import (
	"context"
	"log/slog"
	"sync"

	"unified-backend/internal/domain"
)

// RolePermissionsReader — интерфейс для загрузки permissions из хранилища.
type RolePermissionsReader interface {
	GetAll(ctx context.Context) ([]domain.RolePermissions, error)
}

// PermissionsCache — потокобезопасный in-memory кеш permissions по роли.
type PermissionsCache struct {
	mu    sync.RWMutex
	cache map[string]domain.RolePermissions
	repo  RolePermissionsReader
	admin string
}

func NewPermissionsCache(repo RolePermissionsReader, adminRole string) *PermissionsCache {
	return &PermissionsCache{
		cache: make(map[string]domain.RolePermissions),
		repo:  repo,
		admin: adminRole,
	}
}

// NewStaticPermissionsCache создаёт кеш из заранее заданной мапы.
// Используется в unit-тестах без реальной БД.
func NewStaticPermissionsCache(perms map[string]domain.RolePermissions) *staticPermissionsCache {
	return &staticPermissionsCache{cache: perms}
}

type staticPermissionsCache struct {
	cache map[string]domain.RolePermissions
}

func (s *staticPermissionsCache) Get(role string) domain.RolePermissions {
	if p, ok := s.cache[role]; ok {
		return p
	}
	return domain.RolePermissions{}
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

// Set обновляет одну роль в кеше.
func (c *PermissionsCache) Set(p domain.RolePermissions) {
	c.mu.Lock()
	c.cache[p.Role] = p
	c.mu.Unlock()
}

// GetAll возвращает снимок всего кеша.
func (c *PermissionsCache) GetAll() []domain.RolePermissions {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]domain.RolePermissions, 0, len(c.cache))
	for _, p := range c.cache {
		out = append(out, p)
	}
	return out
}
