package service

import "unified-backend/internal/domain"

// staticPermissionsCache — иммутабельная реализация PermissionsCacheIface для тестов.
type staticPermissionsCache struct {
	data map[string]domain.RolePermissions
}

// NewStaticPermissionsCache создаёт PermissionsCacheIface с заданными значениями.
// Используйте в тестах вместо полного PermissionsCache.
func NewStaticPermissionsCache(perms map[string]domain.RolePermissions) PermissionsCacheIface {
	return &staticPermissionsCache{data: perms}
}

func (s *staticPermissionsCache) Get(role string) domain.RolePermissions {
	if p, ok := s.data[role]; ok {
		return p
	}
	return domain.DefaultUserPermissions(role)
}

func (s *staticPermissionsCache) GetMerged(roles []string) domain.RolePermissions {
	var merged domain.RolePermissions
	for _, r := range roles {
		p := s.Get(r)
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
