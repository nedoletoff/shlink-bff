package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"unified-backend/internal/config"
	"unified-backend/internal/controller"
	"unified-backend/internal/domain"
	"unified-backend/internal/middleware"
	"unified-backend/internal/repository/postgres"
	"unified-backend/internal/service"
)

type SystemHandler struct {
	cfg          *config.Config
	shlinkSvc    *service.ShlinkService
	settingsRepo *postgres.ServerSettingsRepository
	permCtrl     controller.PermChecker
}

func NewSystemHandler(
	cfg *config.Config,
	svc *service.ShlinkService,
	settingsRepo *postgres.ServerSettingsRepository,
	permCtrl controller.PermChecker,
) *SystemHandler {
	return &SystemHandler{cfg: cfg, shlinkSvc: svc, settingsRepo: settingsRepo, permCtrl: permCtrl}
}

type settingsResponse struct {
	ShortCodeLength     int    `json:"shortCodeLength"`
	AllowCustomSlugs    bool   `json:"allowCustomSlugs"`
	UserSlugPrefix      bool   `json:"userSlugPrefix"`
	Domain              string `json:"domain"`
	ShlinkVersion       string `json:"shlinkVersion"`
	Connected           bool   `json:"connected"`
	MaxVisitsDefault    int    `json:"maxVisitsDefault"`
	LinkTtlDefaultDays  int    `json:"linkTtlDefaultDays"`
	AdminRole           string `json:"adminRole"`
	RoleSource          string `json:"roleSource"`
	CorsAllowedOrigins  string `json:"corsAllowedOrigins"`
	ShlinkRunnerMode    string `json:"shlinkRunnerMode"`
	ShlinkContainerName string `json:"shlinkContainerName"`
}

// GET /api/settings – проверяем system.config.view или system.config
func (h *SystemHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "unauthorized"}, http.StatusUnauthorized)
		return
	}
	ok, err := h.permCtrl.Check(r.Context(), user.ID, domain.PermSystemConfigView)
	if err != nil {
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}
	if !ok {
		// fallback: может быть полный доступ
		ok, _ = h.permCtrl.Check(r.Context(), user.ID, domain.PermSystemConfig)
		if !ok {
			writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
			return
		}
	}
	health, err := h.shlinkSvc.Client().GetHealth(r.Context())
	connected := err == nil
	version := ""
	if health != nil {
		version = health.Version
	}
	domain := h.cfg.ShlinkDefaultDomain
	if domain == "" {
		domain = h.cfg.ShlinkInternalURL
	}
	writeJSON(w, settingsResponse{
		ShortCodeLength:     h.cfg.ShlinkShortIDLength,
		AllowCustomSlugs:    h.cfg.UserCustomSlugEnabled,
		UserSlugPrefix:      h.cfg.UserSlugPrefixEnabled,
		Domain:              domain,
		ShlinkVersion:       version,
		Connected:           connected,
		MaxVisitsDefault:    h.cfg.MaxVisitsDefault,
		LinkTtlDefaultDays:  h.cfg.LinkTtlDefaultDays,
		AdminRole:           h.cfg.AdminRole,
		RoleSource:          string(h.cfg.RoleSource),
		CorsAllowedOrigins:  strings.Join(h.cfg.CORSAllowedOrigins, ","),
		ShlinkRunnerMode:    h.cfg.ShlinkRunnerMode,
		ShlinkContainerName: h.cfg.ShlinkContainerName,
	}, http.StatusOK)
}

type patchSettingsPayload struct {
	ShortCodeLength     *int    `json:"shortCodeLength"`
	AllowCustomSlugs    *bool   `json:"allowCustomSlugs"`
	UserSlugPrefix      *bool   `json:"userSlugPrefix"`
	Domain              *string `json:"domain"`
	MaxVisitsDefault    *int    `json:"maxVisitsDefault"`
	LinkTtlDefaultDays  *int    `json:"linkTtlDefaultDays"`
	AdminRole           *string `json:"adminRole"`
	ShlinkRunnerMode    *string `json:"shlinkRunnerMode"`
	ShlinkContainerName *string `json:"shlinkContainerName"`
}

// PATCH /api/settings – требует system.config (полный доступ)
func (h *SystemHandler) PatchSettings(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "unauthorized"}, http.StatusUnauthorized)
		return
	}
	ok, err := h.permCtrl.Check(r.Context(), user.ID, domain.PermSystemConfig)
	if err != nil {
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}
	if !ok {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}

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
	if payload.MaxVisitsDefault != nil {
		h.cfg.MaxVisitsDefault = *payload.MaxVisitsDefault
		dbUpdates["max_visits_default"] = strconv.Itoa(*payload.MaxVisitsDefault)
	}
	if payload.LinkTtlDefaultDays != nil {
		h.cfg.LinkTtlDefaultDays = *payload.LinkTtlDefaultDays
		dbUpdates["link_ttl_default_days"] = strconv.Itoa(*payload.LinkTtlDefaultDays)
	}
	if payload.AdminRole != nil && *payload.AdminRole != "" {
		h.cfg.AdminRole = *payload.AdminRole
		dbUpdates["admin_role"] = *payload.AdminRole
	}
	if payload.ShlinkRunnerMode != nil && *payload.ShlinkRunnerMode != "" {
		h.cfg.ShlinkRunnerMode = *payload.ShlinkRunnerMode
		dbUpdates["shlink_runner_mode"] = *payload.ShlinkRunnerMode
	}
	if payload.ShlinkContainerName != nil && *payload.ShlinkContainerName != "" {
		h.cfg.ShlinkContainerName = *payload.ShlinkContainerName
		dbUpdates["shlink_container_name"] = *payload.ShlinkContainerName
	}

	if h.settingsRepo != nil && len(dbUpdates) > 0 {
		if err := h.settingsRepo.SetMany(r.Context(), dbUpdates, user.Sub); err != nil {
			slog.Error("system_handler: persist settings failed", "err", err)
		}
	}

	if payload.ShortCodeLength != nil {
		if err := h.shlinkSvc.Client().PatchSettings(
			r.Context(), h.cfg.ShlinkAdminAPIKey, h.cfg.ShlinkShortIDLength,
		); err != nil {
			slog.Warn("system_handler: shlink PatchSettings failed", "err", err)
		}
	}

	writeJSON(w, map[string]string{"status": "updated"}, http.StatusOK)
}

