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
	CreateTag(ctx context.Context, apiKey string, body io.Reader) (*shlink.TagsWithStatsResponse, error)
}

// Compile-time check: *shlink.Client реализует ShlinkClientIface.
var _ ShlinkClientIface = (*shlink.Client)(nil)

// PermissionsCacheIface — базовый интерфейс чтения permissions.
type PermissionsCacheIface interface {
	Get(role string) domain.RolePermissions
	GetMerged(roles []string) domain.RolePermissions
}

// PermissionsCacheAdmin — расширенный интерфейс для хендлеров управления ролями.
type PermissionsCacheAdmin interface {
	PermissionsCacheIface
	GetAll() []domain.RolePermissions
	Set(p domain.RolePermissions)
	Delete(role string)
	Reload(ctx context.Context) error
}

// Compile-time check: *PermissionsCache реализует PermissionsCacheAdmin.
var _ PermissionsCacheAdmin = (*PermissionsCache)(nil)

type ShlinkService struct {
	client ShlinkClientIface
	cfg    *config.Config
	perms  PermissionsCacheIface
}

func NewShlinkService(client ShlinkClientIface, cfg *config.Config, perms PermissionsCacheIface) *ShlinkService {
	return &ShlinkService{client: client, cfg: cfg, perms: perms}
}

// resolveAPIKey возвращает shlink api-ключ пользователя или дефолтный из конфига.
func (s *ShlinkService) resolveAPIKey(user *domain.User) string {
	if user != nil && user.ShlinkAPIKey != "" {
		return user.ShlinkAPIKey
	}
	return s.cfg.ShlinkAPIKey
}

func (s *ShlinkService) GetShortURLs(ctx context.Context, user *domain.User, rawQuery string) (*shlink.ShortURLsResponse, error) {
	perms := s.perms.Get(string(user.Role))
	if !perms.CanViewOwnLinks && !perms.CanViewAllLinks {
		return nil, fmt.Errorf("forbidden")
	}
	return s.client.GetShortURLs(ctx, s.resolveAPIKey(user), rawQuery)
}

func (s *ShlinkService) GetShortURL(ctx context.Context, user *domain.User, shortCode string) (*shlink.ShortURL, error) {
	return s.client.GetShortURL(ctx, s.resolveAPIKey(user), shortCode)
}

func (s *ShlinkService) GetShortURLVisits(ctx context.Context, user *domain.User, shortCode, startDate, endDate string, itemsPerPage int) (*shlink.VisitsResponse, error) {
	return s.client.GetShortURLVisits(ctx, s.resolveAPIKey(user), shortCode, startDate, endDate, itemsPerPage)
}

func (s *ShlinkService) CreateShortURL(ctx context.Context, user *domain.User, body io.Reader) (*shlink.ShortURL, error) {
	perms := s.perms.Get(string(user.Role))
	if !perms.CanCreateLinks {
		return nil, fmt.Errorf("forbidden")
	}
	return s.client.CreateShortURL(ctx, s.resolveAPIKey(user), body)
}

func (s *ShlinkService) UpdateShortURL(ctx context.Context, user *domain.User, shortCode string, body io.Reader) (*shlink.ShortURL, error) {
	perms := s.perms.Get(string(user.Role))
	if !perms.CanEditOwnLinks && !perms.CanEditAllLinks {
		return nil, fmt.Errorf("forbidden")
	}
	return s.client.UpdateShortURL(ctx, s.resolveAPIKey(user), shortCode, body)
}

func (s *ShlinkService) DeleteShortURL(ctx context.Context, user *domain.User, shortCode string) error {
	perms := s.perms.Get(string(user.Role))
	if !perms.CanDeleteOwnLinks && !perms.CanDeleteAllLinks {
		return fmt.Errorf("forbidden")
	}
	return s.client.DeleteShortURL(ctx, s.resolveAPIKey(user), shortCode)
}

func (s *ShlinkService) GetTags(ctx context.Context, user *domain.User) (*shlink.TagsWithStatsResponse, error) {
	return s.client.GetTags(ctx, s.resolveAPIKey(user))
}

func (s *ShlinkService) CreateTag(ctx context.Context, user *domain.User, body io.Reader) (*shlink.TagsWithStatsResponse, error) {
	perms := s.perms.Get(string(user.Role))
	if !perms.CanManageOwnTags && !perms.CanManageAllTags {
		return nil, fmt.Errorf("forbidden")
	}
	return s.client.CreateTag(ctx, s.resolveAPIKey(user), body)
}

func (s *ShlinkService) RenameTag(ctx context.Context, user *domain.User, body io.Reader) error {
	perms := s.perms.Get(string(user.Role))
	if !perms.CanManageOwnTags && !perms.CanManageAllTags {
		return fmt.Errorf("forbidden")
	}
	return s.client.RenameTag(ctx, s.resolveAPIKey(user), body)
}

func (s *ShlinkService) DeleteTags(ctx context.Context, user *domain.User, tags []string) error {
	perms := s.perms.Get(string(user.Role))
	if !perms.CanManageOwnTags && !perms.CanManageAllTags {
		return fmt.Errorf("forbidden")
	}
	return s.client.DeleteTags(ctx, s.resolveAPIKey(user), tags)
}

func (s *ShlinkService) GetNonOrphanVisits(ctx context.Context, user *domain.User, startDate, endDate string, itemsPerPage int) (*shlink.VisitsResponse, error) {
	return s.client.GetNonOrphanVisits(ctx, s.resolveAPIKey(user), startDate, endDate, itemsPerPage)
}

func (s *ShlinkService) PatchSettings(ctx context.Context, shortCodeLength int) error {
	return s.client.PatchSettings(ctx, s.cfg.ShlinkAPIKey, shortCodeLength)
}

func (s *ShlinkService) GetHealth(ctx context.Context) (*shlink.HealthResponse, error) {
	return s.client.GetHealth(ctx)
}

// BuildSlug формирует slug с учётом prefix пользователя.
func (s *ShlinkService) BuildSlug(user *domain.User, slug string) string {
	if user == nil || user.SlugPrefix == "" {
		return slug
	}
	if slug == "" {
		return user.SlugPrefix
	}
	return strings.Join([]string{user.SlugPrefix, slug}, "-")
}
