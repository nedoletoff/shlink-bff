// Удалён: логика перенесена в system_handler.go.
package handler

// ApplyDBOverrides — устаревшая заглушка, сохранена для обратной совместимости с main.go.
// Deprecated: используйте ServerSettingsRepository.ApplyAll.
import (
	"strconv"
	"unified-backend/internal/config"
)

func ApplyDBOverrides(cfg *config.Config, settings map[string]string) {
	if v, ok := settings["short_code_length"]; ok {
		if n, err := strconv.Atoi(v); err == nil && n >= 3 && n <= 32 {
			cfg.ShlinkShortIDLength = n
		}
	}
	if v, ok := settings["allow_custom_slugs"]; ok {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.UserCustomSlugEnabled = b
		}
	}
	if v, ok := settings["user_slug_prefix"]; ok {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.UserSlugPrefixEnabled = b
		}
	}
	if v, ok := settings["default_domain"]; ok && v != "" {
		cfg.ShlinkDefaultDomain = v
	}
}
