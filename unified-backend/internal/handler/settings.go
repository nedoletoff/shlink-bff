package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"unified-backend/internal/config"
	"unified-backend/internal/service"
)

// SettingsHandler обрабатывает GET/PATCH /api/admin/settings
type SettingsHandler struct {
	cfg       *config.Config
	shlinkSvc *service.ShlinkService
}

func NewSettingsHandler(cfg *config.Config, svc *service.ShlinkService) *SettingsHandler {
	return &SettingsHandler{cfg: cfg, shlinkSvc: svc}
}

type settingsResponse struct {
	ShortCodeLength   int    `json:"shortCodeLength"`
	AllowCustomSlugs  bool   `json:"allowCustomSlugs"`
	UserSlugPrefix    bool   `json:"userSlugPrefix"`
	// Domain — публичный домен (по умолчанию из SHLINK_DEFAULT_DOMAIN, не внутренний адрес API)
	Domain            string `json:"domain"`
	ShlinkVersion     string `json:"shlinkVersion"`
	Connected         bool   `json:"connected"`
}

// GET /api/admin/settings
func (h *SettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	health, err := h.shlinkSvc.Client().GetHealth(r.Context())
	connected := err == nil
	version := ""
	if health != nil {
		version = health.Version
	}

	// Используем SHLINK_DEFAULT_DOMAIN если задан, иначе fallback на ShlinkURL
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

// PATCH /api/admin/settings
// Обновляет runtime feature-флаги в памяти (до рестарта).
func (h *SettingsHandler) PatchSettings(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
	if err != nil {
		writeJSON(w, map[string]string{"error": "bad request"}, http.StatusBadRequest)
		return
	}
	var payload struct {
		ShortCodeLength  *int    `json:"shortCodeLength"`
		AllowCustomSlugs *bool   `json:"allowCustomSlugs"`
		UserSlugPrefix   *bool   `json:"userSlugPrefix"`
		Domain           *string `json:"domain"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		writeJSON(w, map[string]string{"error": "invalid json"}, http.StatusBadRequest)
		return
	}
	if payload.ShortCodeLength != nil {
		v := *payload.ShortCodeLength
		if v < 3 {
			v = 3
		}
		if v > 32 {
			v = 32
		}
		h.cfg.ShlinkShortIDLength = v
	}
	if payload.AllowCustomSlugs != nil {
		h.cfg.UserCustomSlugEnabled = *payload.AllowCustomSlugs
	}
	if payload.UserSlugPrefix != nil {
		h.cfg.UserSlugPrefixEnabled = *payload.UserSlugPrefix
	}
	if payload.Domain != nil && *payload.Domain != "" {
		h.cfg.ShlinkDefaultDomain = *payload.Domain
	}
	writeJSON(w, map[string]string{"status": "updated"}, http.StatusOK)
}
