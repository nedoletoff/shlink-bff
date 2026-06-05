package test

import (
	"unified-backend/internal/domain"
	"unified-backend/internal/service"
)

// newTestPermissionsCache returns a PermissionsCache pre-loaded with default
// admin and user permissions. Used by handler_me_test.go and similar tests
// that need a cache without a DB-backed repo.
func newTestPermissionsCache() *service.PermissionsCache {
	cache := service.NewPermissionsCache(nil, domain.RoleAdmin)
	cache.Set(domain.DefaultAdminPermissions(domain.RoleAdmin))
	cache.Set(domain.DefaultUserPermissions(domain.RoleUser))
	return cache
}
