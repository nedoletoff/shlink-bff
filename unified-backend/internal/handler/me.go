package handler

import (
	"log/slog"
	"net/http"

	"unified-backend/internal/config"
	"unified-backend/internal/middleware"
	"unified-backend/internal/service"
)

type MeResponse struct {
	Sub         string       `json:"sub"`
	Username    string       `json:"username"`
	Email       string       `json:"email"`
	Role        string       `json:"role"`
	Permissions []string     `json:"permissions"`
	HasAPIKey   bool         `json:"hasApiKey"`
	Features    FeatureFlags `json:"features"`
	SlugPrefix  string       `json:"slugPrefix,omitempty"`
}

type FeatureFlags struct {
	UserSlugPrefixEnabled    bool `json:"userSlugPrefixEnabled"`
	UserTagInternalIdEnabled bool `json:"userTagInternalIdEnabled"`
}

type MeHandler struct {
	cfg     *config.Config
	permSvc *service.PermissionService
}

func NewMeHandler(cfg *config.Config, permSvc *service.PermissionService) *MeHandler {
	return &MeHandler{cfg: cfg, permSvc: permSvc}
}

func (h *MeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		slog.Error("me: user not in context")
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}

	perms, err := h.permSvc.GetUserPermissions(r.Context(), user.ID)
	if err != nil {
		slog.Error("me: failed to get permissions", "sub", user.Sub, "err", err)
	}
	// Гарантируем []string{} вместо null в JSON если пермишни пустые или ошибка.
	if perms == nil {
		perms = []string{}
	}

	resp := MeResponse{
		Sub:         user.Sub,
		Username:    user.Username,
		Email:       user.Email,
		Role:        user.Role,
		Permissions: perms,
		HasAPIKey:   user.ShlinkAPIKey != "",
		Features: FeatureFlags{
			UserSlugPrefixEnabled:    h.cfg.UserSlugPrefixEnabled,
			UserTagInternalIdEnabled: h.cfg.UserTagInternalIdEnabled,
		},
	}

	if h.cfg.UserSlugPrefixEnabled && user.SlugPrefix != "" {
		resp.SlugPrefix = user.SlugPrefix
	}

	writeJSON(w, resp, http.StatusOK)
}
