package handler

import (
	"log/slog"
	"net/http"

	"unified-backend/internal/config"
	"unified-backend/internal/middleware"
)

type MeResponse struct {
	Sub         string          `json:"sub"`
	Username    string          `json:"username"`
	Email       string          `json:"email"`
	Role        string          `json:"role"`
	Permissions map[string]bool `json:"permissions"`
	HasAPIKey   bool            `json:"hasApiKey"`
	Features    FeatureFlags    `json:"features"`
	SlugPrefix  string          `json:"slugPrefix,omitempty"`
}

type FeatureFlags struct {
	UserSlugPrefixEnabled    bool `json:"userSlugPrefixEnabled"`
	UserTagInternalIdEnabled bool `json:"userTagInternalIdEnabled"`
}

type MeHandler struct {
	cfg *config.Config
}

func NewMeHandler(cfg *config.Config) *MeHandler {
	return &MeHandler{cfg: cfg}
}

func (h *MeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// user уже загружен из БД в RequireActiveUser middleware
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		slog.Error("me: user not in context")
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}

	perms := user.ComputePermissions()

	resp := MeResponse{
		Sub:      user.Sub,
		Username: user.Username,
		Email:    user.Email,
		Role:     string(user.Role),
		Permissions: map[string]bool{
			"canCreateShortUrl": perms.CanCreateShortURL,
			"canEditOwnLinks":   perms.CanEditOwnLinks,
			"canDeleteOwnLinks": perms.CanDeleteOwnLinks,
			"canManageOwnTags":  perms.CanManageOwnTags,
			"canViewAuditLogs":  perms.CanViewAuditLogs,
			"canManageUsers":    perms.CanManageUsers,
		},
		// Только флаг наличия — реальный ключ НИКОГДА не покидает backend
		HasAPIKey: user.ShlinkAPIKey != "",
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


