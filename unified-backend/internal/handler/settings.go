package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"unified-backend/internal/config"
	"unified-backend/internal/middleware"
	"unified-backend/internal/repository/postgres"
	"unified-backend/internal/service"
)

// SettingsHandler обрабатывает GET/PATCH /api/settings (и /api/admin/settings).
type SettingsHandler struct {
	cfg         *config.Config
	shlinkSvc   *service.ShlinkService
	settingsRepo *postgres.ServerSettingsRepository
}

func NewSettingsHandler(
	cfg *config.Config,
	svc *service.ShlinkService,
	settingsRepo *postgres.ServerSettingsRepository,
) *SettingsHandler {
	return &SettingsHandler{cfg: cfg, shlinkSvc: svc, settingsRepo: settingsRepo}
}

type settingsResponse struct {
	ShortCodeLength  int    `json:"shortCodeLength"`
	AllowCustomSlugs bool   `json:"allowCustomSlugs"`
	UserSlugPrefix   bool   `json:"userSlugPrefix"`
	Domain           string `json:"domain"`
	ShlinkVersion    string `json:"shlinkVersion"`
	Connected        bool   `json:"connected"`
}

// GET /api/settings  (и /api/admin/settings)
func (h *SettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	health, err := h.shlinkSvc.Client().GetHealth(r.Context())
	connected := err == nil
	version := ""
	if health != nil {
		version = health.Version
	}

	domain := h.cfg.ShlinkDefaultDomain
	if domain == "" {
		domain = h.cfg.ShlinkURL
	}

	writeJSON(w, settingsResponse{
		ShortCodeLength:  h.cfg.ShlinkShortIDLength,
		AllowCustomSlugs: h.cfg.UserCustomSlugEnabled,
		UserSlugPrefix:   h.cfg.UserSlugPrefixEnabled,
		Domain:           domain,
		ShlinkVersion:    version,
		Connected:        connected,
	}, http.StatusOK)
}

type patchSettingsPayload struct {
	ShortCodeLength  *int    `json:"shortCodeLength"`
	AllowCustomSlugs *bool   `json:"allowCustomSlugs"`
	UserSlugPrefix   *bool   `json:"userSlugPrefix"`
	Domain           *string `json:"domain"`
}

// PATCH /api/settings  (и /api/admin/settings)
// 1. Сохраняет в server_settings (персистентно).
// 2. Применяет к live-конфигу (in-memory).
// 3. Для shortCodeLength — проксирует PATCH /rest/v3/settings в shlink.
func (h *SettingsHandler) PatchSettings(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
	if err != nil {
		writeJSON(w, map[string]string{"error": "bad request"}, http.StatusBadRequest)
		return
	}

	var payload patchSettingsPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		writeJSON(w, map[string]string{"error": "invalid json"}, http.StatusBadRequest)
		return
	}

	// ── 1. Собираем изменения для БД и применяем к cfg ────────────────────────
	dbUpdates := make(map[string]string)

	if payload.ShortCodeLength != nil {
		v := *payload.ShortCodeLength
		if v < 3 {
			v = 3
		}
		if v > 32 {
			v = 32
		}
		h.cfg.ShlinkShortIDLength = v
		dbUpdates["short_code_length"] = strconv.Itoa(v)
	}
	if payload.AllowCustomSlugs != nil {
		h.cfg.UserCustomSlugEnabled = *payload.AllowCustomSlugs
		dbUpdates["allow_custom_slugs"] = strconv.FormatBool(*payload.AllowCustomSlugs)
	}
	if payload.UserSlugPrefix != nil {
		h.cfg.UserSlugPrefixEnabled = *payload.UserSlugPrefix
		dbUpdates["user_slug_prefix"] = strconv.FormatBool(*payload.UserSlugPrefix)
	}
	if payload.Domain != nil && *payload.Domain != "" {
		h.cfg.ShlinkDefaultDomain = *payload.Domain
		dbUpdates["default_domain"] = *payload.Domain
	}

	// ── 2. Персистируем в БД ──────────────────────────────────────────────────
	if h.settingsRepo != nil && len(dbUpdates) > 0 {
		updatedBy := ""
		if user != nil {
			updatedBy = user.Sub
		}
		if err := h.settingsRepo.SetMany(r.Context(), dbUpdates, updatedBy); err != nil {
			slog.Error("settings: persist failed", "err", err)
			// Не фатально — cfg уже обновлён, ответ не прерываем
		}
	}

	// ── 3. Применяем shortCodeLength в shlink через PATCH /rest/v3/settings ──
	if payload.ShortCodeLength != nil {
		adminKey := h.cfg.ShlinkAdminAPIKey
		if err := h.shlinkSvc.Client().PatchSettings(
			r.Context(),
			adminKey,
			h.cfg.ShlinkShortIDLength,
		); err != nil {
			slog.Warn("settings: shlink PATCH /settings failed", "err", err,
				"shortCodeLength", h.cfg.ShlinkShortIDLength)
			// Не фатально: наш cfg обновлён, shlink может не поддерживать endpoint
		}
	}

	writeJSON(w, map[string]string{"status": "updated"}, http.StatusOK)
}

// ApplyDBOverrides загружает значения из БД и переопределяет cfg.
// Вызывается из main.go после MustLoad(), если config_source=db.
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
	_ = fmt.Sprintf // keep import
}
