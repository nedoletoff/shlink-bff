package service

import (
	"context"
	"fmt"
	"strings"

	"unified-backend/internal/config"
	"unified-backend/internal/domain"
	"unified-backend/internal/shlink"
)

type ShlinkService struct {
	client *shlink.Client
	cfg    *config.Config
}

func NewShlinkService(client *shlink.Client, cfg *config.Config) *ShlinkService {
	return &ShlinkService{client: client, cfg: cfg}
}

// EnforceSlugPrefix добавляет/валидирует prefix для роли user.
// Для admin — пропускает без изменений.
// Возвращает итоговый slug (может быть пустым → Shlink генерирует сам).
func (s *ShlinkService) EnforceSlugPrefix(
	ctx context.Context,
	user *domain.User,
	customSlug *string,
) (string, error) {
	if !s.cfg.UserSlugPrefixEnabled || user.Role == domain.RoleAdmin {
		if customSlug != nil {
			return *customSlug, nil
		}
		return "", nil
	}

	prefix := user.SlugPrefix
	if prefix == "" {
		return "", fmt.Errorf("user %s has no slug prefix configured", user.Sub)
	}

	if customSlug == nil || *customSlug == "" {
		// Пустой slug → вернём только префикс, Shlink добавит суффикс
		return prefix, nil
	}

	slug := *customSlug
	if !strings.HasPrefix(slug, prefix) {
		return "", fmt.Errorf("slug must start with prefix %q", prefix)
	}
	return slug, nil
}

// FilterShortURLsByUser фильтрует ссылки по slug_prefix для роли user.
// Работает только если feature flag включён.
func (s *ShlinkService) FilterShortURLsByUser(
	urls []shlink.ShortURL,
	user *domain.User,
) []shlink.ShortURL {
	if !s.cfg.UserSlugPrefixEnabled || user.Role == domain.RoleAdmin {
		return urls
	}
	prefix := user.SlugPrefix
	if prefix == "" {
		return urls
	}
	filtered := make([]shlink.ShortURL, 0, len(urls))
	for _, u := range urls {
		if strings.HasPrefix(u.ShortCode, prefix) {
			filtered = append(filtered, u)
		}
	}
	return filtered
}

// CanModifyShortCode проверяет, вправе ли пользователь изменять/удалять ссылку с данным shortCode.
//
// Модель владения здесь основана на slug-префиксе (так же, как Filter/Enforce):
//   - admin — может всё;
//   - при выключенном feature flag изоляция не применяется;
//   - role=user с префиксом — только свои ссылки (shortCode начинается с префикса).
//
// Без этой проверки пользователь мог бы изменить/удалить чужую ссылку, зная её shortCode (#8).
func (s *ShlinkService) CanModifyShortCode(user *domain.User, shortCode string) bool {
	if user.Role == domain.RoleAdmin {
		return true
	}
	if !s.cfg.UserSlugPrefixEnabled {
		return true
	}
	if user.SlugPrefix == "" {
		// feature включён, но префикс не задан — изменять нечего безопасного, запрещаем.
		return false
	}
	return strings.HasPrefix(shortCode, user.SlugPrefix)
}

// Client exposes the shlink client for use in handlers
func (s *ShlinkService) Client() *shlink.Client {
	return s.client
}
