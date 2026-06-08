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
