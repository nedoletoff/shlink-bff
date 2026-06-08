package service

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"unified-backend/internal/config"
	"unified-backend/internal/domain"
	"unified-backend/internal/shlink"
)

// ShlinkClientIface — интерфейс Shlink HTTP-клиента.
// Определён здесь чтобы позволить подменять stub в unit-тестах без реального Shlink.
type ShlinkClientIface interface {
	GetShortURLs(ctx context.Context, apiKey, rawQuery string) (*shlink.ShortURLsResponse, error)
	GetShortURL(ctx context.Context, apiKey, shortCode string) (*shlink.ShortURL, error)
	GetShortURLVisits(ctx context.Context, apiKey, shortCode, startDate, endDate string, itemsPerPage int) (*shlink.VisitsResponse, error)
	CreateShortURL(ctx context.Context, apiKey string, body io.Reader) (*shlink.ShortURL, error)
	UpdateShortURL(ctx context.Context, apiKey, shortCode string, body io.Reader) (*shlink.ShortURL, error)
	DeleteShortURL(ctx context.Context, apiKey, shortCode string) error
	GetTags(ctx context.Context, apiKey string) (*shlink.TagsWithStatsResponse, error)
	RenameTag(ctx context.Context, apiKey string, body io.Reader) error
	DeleteTags(ctx context.Context, apiKey string, tags []string) error
	GetNonOrphanVisits(ctx context.Context, apiKey, startDate, endDate string, itemsPerPage int) (*shlink.VisitsResponse, error)
	PatchSettings(ctx context.Context, adminAPIKey string, shortCodeLength int) error
	GetHealth(ctx context.Context) (*shlink.HealthResponse, error)
	ValidateVersion(ctx context.Context, minMajor int, attempts int, delay time.Duration) error
}

// Compile-time check: *shlink.Client реализует ShlinkClientIface.
var _ ShlinkClientIface = (*shlink.Client)(nil)

type ShlinkService struct {
	client ShlinkClientIface
	cfg    *config.Config
	perms  PermissionsCacheIface
}

// PermissionsCacheIface позволяет подменять кеш в тестах.
type PermissionsCacheIface interface {
	Get(role string) domain.RolePermissions
}

func NewShlinkService(client ShlinkClientIface, cfg *config.Config, perms PermissionsCacheIface) *ShlinkService {
	return &ShlinkService{client: client, cfg: cfg, perms: perms}
}

// Perms возвращает permissions для роли пользователя.
func (s *ShlinkService) Perms(user *domain.User) domain.RolePermissions {
	return s.perms.Get(string(user.Role))
}

// EnforceSlugPrefix валидирует/устанавливает slug с учётом permissions.
func (s *ShlinkService) EnforceSlugPrefix(
	ctx context.Context,
	user *domain.User,
	customSlug *string,
) (string, error) {
	p := s.perms.Get(string(user.Role))

	if !p.CanCreateLinks {
		return "", fmt.Errorf("role %q is not allowed to create links", user.Role)
	}

	hasCustomSlug := customSlug != nil && *customSlug != ""

	if hasCustomSlug {
		if !p.CanViewAllLinks && !s.cfg.UserCustomSlugEnabled {
			return "", fmt.Errorf("custom slugs are disabled for role %q", user.Role)
		}
		if !p.CanCreateWithCustomSlug {
			return "", fmt.Errorf("role %q is not allowed to set a custom slug", user.Role)
		}
	}

	if !hasCustomSlug && !p.CanCreateWithoutSlug {
		return "", fmt.Errorf("role %q must provide a custom slug", user.Role)
	}

	if !s.cfg.UserSlugPrefixEnabled || p.CanViewAllLinks {
		if hasCustomSlug {
			return *customSlug, nil
		}
		return "", nil
	}

	prefix := user.SlugPrefix
	if prefix == "" {
		return "", fmt.Errorf("user %s has no slug prefix configured", user.Sub)
	}

	if !hasCustomSlug {
		return prefix, nil
	}

	slug := *customSlug
	if !strings.HasPrefix(slug, prefix) {
		return "", fmt.Errorf("slug must start with prefix %q", prefix)
	}
	return slug, nil
}

// FilterShortURLsByUser фильтрует ссылки согласно permissions роли.
func (s *ShlinkService) FilterShortURLsByUser(
	urls []shlink.ShortURL,
	user *domain.User,
	ownedCodes map[string]struct{},
) []shlink.ShortURL {
	p := s.perms.Get(string(user.Role))

	if p.CanViewAllLinks {
		return urls
	}
	if !p.CanViewOwnLinks {
		return []shlink.ShortURL{}
	}

	filtered := make([]shlink.ShortURL, 0, len(ownedCodes))
	for _, u := range urls {
		if _, ok := ownedCodes[u.ShortCode]; ok {
			filtered = append(filtered, u)
		}
	}
	return filtered
}

// CanModifyShortCodeByPerms проверяет права роли на edit/delete.
func (s *ShlinkService) CanModifyShortCodeByPerms(user *domain.User, isDelete bool) (canAll bool, canOwn bool) {
	p := s.perms.Get(string(user.Role))
	if isDelete {
		return p.CanDeleteAllLinks, p.CanDeleteOwnLinks
	}
	return p.CanEditAllLinks, p.CanEditOwnLinks
}

// Client возвращает shlink-клиент для хендлеров.
func (s *ShlinkService) Client() ShlinkClientIface {
	return s.client
}

// ShlinkShortIDLength возвращает сконфигурированную длину short ID.
func (s *ShlinkService) ShlinkShortIDLength() int {
	return s.cfg.ShlinkShortIDLength
}
