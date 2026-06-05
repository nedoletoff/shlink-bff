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
	perms  *PermissionsCache
}

func NewShlinkService(client *shlink.Client, cfg *config.Config, perms *PermissionsCache) *ShlinkService {
	return &ShlinkService{client: client, cfg: cfg, perms: perms}
}

// Perms возвращает permissions для роли пользователя.
func (s *ShlinkService) Perms(user *domain.User) domain.RolePermissions {
	return s.perms.Get(user.Role)
}

// EnforceSlugPrefix валидирует/устанавливает slug с учётом permissions.
//
// Логика:
//  1. Если у роли нет can_create_links → ошибка.
//  2. Если передан customSlug:
//     a. Если UserCustomSlugEnabled=false и роль не admin (CanViewAllLinks) → ошибка.
//     b. Если нет can_create_with_custom_slug → ошибка.
//  3. Если не передан slug и нет can_create_without_slug → ошибка.
//  4. Если включён UserSlugPrefixEnabled и роль не имеет глобальных прав
//     → принудительно добавляем slug_prefix.
func (s *ShlinkService) EnforceSlugPrefix(
	ctx context.Context,
	user *domain.User,
	customSlug *string,
) (string, error) {
	p := s.perms.Get(user.Role)

	if !p.CanCreateLinks {
		return "", fmt.Errorf("role %q is not allowed to create links", user.Role)
	}

	hasCustomSlug := customSlug != nil && *customSlug != ""

	// Двухступенчатая проверка кастомного слага.
	if hasCustomSlug {
		// Ступень 1: feature-флаг. Не распространяется на admin (CanViewAllLinks).
		if !p.CanViewAllLinks && !s.cfg.UserCustomSlugEnabled {
			return "", fmt.Errorf("custom slugs are disabled for role %q", user.Role)
		}
		// Ступень 2: permission роли.
		if !p.CanCreateWithCustomSlug {
			return "", fmt.Errorf("role %q is not allowed to set a custom slug", user.Role)
		}
	}

	if !hasCustomSlug && !p.CanCreateWithoutSlug {
		return "", fmt.Errorf("role %q must provide a custom slug", user.Role)
	}

	// Принудительный slug_prefix только если изоляция включена
	// и роль не имеет глобального доступа к чужим ссылкам.
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
// Фильтрация по ownership (конкретные short_code) должна была быть выполнена до вызова этого метода:
// хендлер должен передать только ownedCodes (сет short_code из url_ownership).
func (s *ShlinkService) FilterShortURLsByUser(
	urls []shlink.ShortURL,
	user *domain.User,
	ownedCodes map[string]struct{},
) []shlink.ShortURL {
	p := s.perms.Get(user.Role)

	if p.CanViewAllLinks {
		return urls
	}
	if !p.CanViewOwnLinks {
		return []shlink.ShortURL{}
	}

	// Фильтрация по явной таблице ownership.
	filtered := make([]shlink.ShortURL, 0, len(ownedCodes))
	for _, u := range urls {
		if _, ok := ownedCodes[u.ShortCode]; ok {
			filtered = append(filtered, u)
		}
	}
	return filtered
}

// CanModifyShortCodeByPerms проверяет права роли на edit/delete.
// Не проверяет ownership — это делает хендлер через ownerRepo.IsOwner.
func (s *ShlinkService) CanModifyShortCodeByPerms(user *domain.User, isDelete bool) (canAll bool, canOwn bool) {
	p := s.perms.Get(user.Role)
	if isDelete {
		return p.CanDeleteAllLinks, p.CanDeleteOwnLinks
	}
	return p.CanEditAllLinks, p.CanEditOwnLinks
}

// Client возвращает shlink-клиент для хендлеров.
func (s *ShlinkService) Client() *shlink.Client {
	return s.client
}

// ShlinkShortIDLength возвращает сконфигурированную длину short ID (0 = не задана).
func (s *ShlinkService) ShlinkShortIDLength() int {
	return s.cfg.ShlinkShortIDLength
}
