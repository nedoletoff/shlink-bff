package test

import (
	"unified-backend/internal/config"
	"unified-backend/internal/domain"
	"unified-backend/internal/service"
	"unified-backend/internal/shlink"
)

// newShlinkService creates a ShlinkService with the provided role permissions
// preloaded in the PermissionsCache. The shlink Client is nil — pure logic tests
// must not call methods that reach the network.
func newShlinkService(perms domain.RolePermissions) *service.ShlinkService {
	var nilClient *shlink.Client
	cfg := &config.Config{
		UserSlugPrefixEnabled: true,
		UserCustomSlugEnabled: true,
		ShlinkShortIDLength:   6,
	}
	cache := service.NewPermissionsCache(nil)
	cache.Set(perms)
	return service.NewShlinkService(nilClient, cfg, cache)
}
