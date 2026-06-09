package service

import (
	"context"
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
	CreateTag(ctx context.Context, apiKey string, body io.Reader) (*shlink.TagsWithStatsResponse, error)
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
}

// NewShlinkService создаёт ShlinkService.
// Все проверки прав выполняются через PermissionController в хендлерах.
func NewShlinkService(client ShlinkClientIface, cfg *config.Config) *ShlinkService {
	return &ShlinkService{client: client, cfg: cfg}
}

// Client возвращает сырой клиент.
func (s *ShlinkService) Client() ShlinkClientIface {
	return s.client
}

// resolveAPIKey возвращает shlink api-ключ пользователя или дефолтный из конфига.
func (s *ShlinkService) resolveAPIKey(user *domain.User) string {
	if user != nil && user.ShlinkAPIKey != "" {
		return user.ShlinkAPIKey
	}
	return s.cfg.ShlinkAPIKey
}

func (s *ShlinkService) GetShortURLs(ctx context.Context, user *domain.User, rawQuery string) (*shlink.ShortURLsResponse, error) {
	return s.client.GetShortURLs(ctx, s.resolveAPIKey(user), rawQuery)
}

func (s *ShlinkService) GetShortURL(ctx context.Context, user *domain.User, shortCode string) (*shlink.ShortURL, error) {
	return s.client.GetShortURL(ctx, s.resolveAPIKey(user), shortCode)
}

func (s *ShlinkService) GetShortURLVisits(ctx context.Context, user *domain.User, shortCode, startDate, endDate string, itemsPerPage int) (*shlink.VisitsResponse, error) {
	return s.client.GetShortURLVisits(ctx, s.resolveAPIKey(user), shortCode, startDate, endDate, itemsPerPage)
}

func (s *ShlinkService) CreateShortURL(ctx context.Context, user *domain.User, body io.Reader) (*shlink.ShortURL, error) {
	return s.client.CreateShortURL(ctx, s.resolveAPIKey(user), body)
}

func (s *ShlinkService) UpdateShortURL(ctx context.Context, user *domain.User, shortCode string, body io.Reader) (*shlink.ShortURL, error) {
	return s.client.UpdateShortURL(ctx, s.resolveAPIKey(user), shortCode, body)
}

func (s *ShlinkService) DeleteShortURL(ctx context.Context, user *domain.User, shortCode string) error {
	return s.client.DeleteShortURL(ctx, s.resolveAPIKey(user), shortCode)
}

func (s *ShlinkService) GetTags(ctx context.Context, user *domain.User) (*shlink.TagsWithStatsResponse, error) {
	return s.client.GetTags(ctx, s.resolveAPIKey(user))
}

func (s *ShlinkService) RenameTag(ctx context.Context, user *domain.User, body io.Reader) error {
	return s.client.RenameTag(ctx, s.resolveAPIKey(user), body)
}

func (s *ShlinkService) DeleteTags(ctx context.Context, user *domain.User, tags []string) error {
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

// FilterShortURLsByUser фильтрует URL по владельцу.
func (s *ShlinkService) FilterShortURLsByUser(urls []shlink.ShortURL, _ *domain.User, ownedCodes map[string]struct{}) []shlink.ShortURL {
	result := make([]shlink.ShortURL, 0)
	for _, u := range urls {
		if _, ok := ownedCodes[u.ShortCode]; ok {
			result = append(result, u)
		}
	}
	return result
}

// EnforceSlugPrefix проверяет и применяет prefix-правила.
func (s *ShlinkService) EnforceSlugPrefix(_ context.Context, user *domain.User, customSlug *string) (string, error) {
	if customSlug == nil || *customSlug == "" {
		return "", nil
	}
	if user == nil || user.SlugPrefix == "" {
		return *customSlug, nil
	}
	prefix := user.SlugPrefix + "-"
	if strings.HasPrefix(*customSlug, prefix) {
		return *customSlug, nil
	}
	return prefix + *customSlug, nil
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
