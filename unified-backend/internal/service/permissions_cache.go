package service

import (
	"context"
	"sync"

	"unified-backend/internal/domain"
)

// RolesRepo — интерфейс хранилища ролей.
type RolesRepo interface {
	GetAll(ctx context.Context) ([]domain.RolePermissions, error)
	Upsert(ctx context.Context, p *domain.RolePermissions) error
}

// PermissionsCache — потокобезопасный кэш permissions ролей.
type PermissionsCache struct {
	mu        sync.RWMutex
	data      map[string]domain.RolePermissions
	repo      RolesRepo
	adminRole string
}

func NewPermissionsCache(repo RolesRepo, adminRole string) *PermissionsCache {
	return &PermissionsCache{
		data:      make(map[string]domain.RolePermissions),
		repo:      repo,
		adminRole: adminRole,
	}
}

// Load загружает все роли из БД в кэш.
func (c *PermissionsCache) Load(ctx context.Context) error {
	perms, err := c.repo.GetAll(ctx)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, p := range perms {
		c.data[p.Role] = p
	}
	return nil
}

// Reload сбрасывает кэш и загружает все роли из БД заново.
func (c *PermissionsCache) Reload(ctx context.Context) error {
	perms, err := c.repo.GetAll(ctx)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]domain.RolePermissions, len(perms))
	for _, p := range perms {
		c.data[p.Role] = p
	}
	return nil
}

// Get возвращает permissions для роли. Если нет — fallback.
func (c *PermissionsCache) Get(role string) domain.RolePermissions {
	c.mu.RLock()
	p, ok := c.data[role]
	c.mu.RUnlock()
	if ok {
		return p
	}
	if role == c.adminRole {
		return domain.DefaultAdminPermissions(role)
	}
	return domain.DefaultUserPermissions(role)
}

// GetMerged возвращает OR-объединение нескольких ролей.
func (c *PermissionsCache) GetMerged(roles []string) domain.RolePermissions {
	var merged domain.RolePermissions
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
		merged.CanDeactivateOwnLinks = merged.CanDeactivateOwnLinks || p.CanDeactivateOwnLinks
		merged.CanDeactivateAllLinks = merged.CanDeactivateAllLinks || p.CanDeactivateAllLinks
		merged.CanReactivateOwnLinks = merged.CanReactivateOwnLinks || p.CanReactivateOwnLinks
		merged.CanReactivateAllLinks = merged.CanReactivateAllLinks || p.CanReactivateAllLinks
		merged.CanDeleteOwnLinksPermanently = merged.CanDeleteOwnLinksPermanently || p.CanDeleteOwnLinksPermanently
		merged.CanDeleteAllLinksPermanently = merged.CanDeleteAllLinksPermanently || p.CanDeleteAllLinksPermanently
		merged.CanManageOwnTags = merged.CanManageOwnTags || p.CanManageOwnTags
		merged.CanManageAllTags = merged.CanManageAllTags || p.CanManageAllTags
		merged.CanViewOwnStats = merged.CanViewOwnStats || p.CanViewOwnStats
		merged.CanViewAllStats = merged.CanViewAllStats || p.CanViewAllStats
		merged.CanViewAuditLogs = merged.CanViewAuditLogs || p.CanViewAuditLogs
		merged.CanManageUsers = merged.CanManageUsers || p.CanManageUsers
		merged.CanManageRoles = merged.CanManageRoles || p.CanManageRoles
		merged.CanManageSettings = merged.CanManageSettings || p.CanManageSettings
	}
	return merged
}

// Set обновляет запись в кэше.
func (c *PermissionsCache) Set(p domain.RolePermissions) {
	c.mu.Lock()
	c.data[p.Role] = p
	c.mu.Unlock()
}

// Delete удаляет роль из кэша.
func (c *PermissionsCache) Delete(role string) {
	c.mu.Lock()
	delete(c.data, role)
	c.mu.Unlock()
}

// GetAll возвращает снимок всех ролей.
func (c *PermissionsCache) GetAll() []domain.RolePermissions {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]domain.RolePermissions, 0, len(c.data))
	for _, p := range c.data {
		out = append(out, p)
	}
	return out
}
