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
//  2. Если передан customSlug и нет can_create_with_custom_slug → ошибка.
//  3. Если не передан slug и нет can_create_without_slug → ошибка.
//  4. Если включён UserSlugPrefixEnabled и роль не may_view_all_links (не имеет глобальных прав)
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

	if hasCustomSlug && !p.CanCreateWithCustomSlug {
		return "", fmt.Errorf("role %q is not allowed to set a custom slug", user.Role)
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
func (s *ShlinkService) FilterShortURLsByUser(
	urls []shlink.ShortURL,
	user *domain.User,
) []shlink.ShortURL {
	p := s.perms.Get(user.Role)

	if p.CanViewAllLinks {
		return urls
	}
	if !p.CanViewOwnLinks {
		return []shlink.ShortURL{}
	}

	// Изоляция по slug_prefix если включена
	if !s.cfg.UserSlugPrefixEnabled {
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

// CanModifyShortCode — edit/delete права с учётом permissions.
func (s *ShlinkService) CanModifyShortCode(user *domain.User, shortCode string, isDelete bool) bool {
	p := s.perms.Get(user.Role)

	if isDelete {
		if p.CanDeleteAllLinks {
			return true
		}
		if !p.CanDeleteOwnLinks {
			return false
		}
	} else {
		if p.CanEditAllLinks {
			return true
		}
		if !p.CanEditOwnLinks {
			return false
		}
	}

	if !s.cfg.UserSlugPrefixEnabled {
		return true
	}
	if user.SlugPrefix == "" {
		return false
	}
	return strings.HasPrefix(shortCode, user.SlugPrefix)
}

// Client возвращает shlink-клиент для хендлеров.
func (s *ShlinkService) Client() *shlink.Client {
	return s.client
}
