package handler

import (
	"encoding/json"
	"net/http"
	"unified-backend/internal/middleware"
	"unified-backend/internal/service"
)

type MeHandler struct {
	permSvc *service.PermissionService
}

func NewMeHandler(permSvc *service.PermissionService) *MeHandler {
	return &MeHandler{permSvc: permSvc}
}

type meResponse struct {
	ID             string   `json:"id"`
	Sub            string   `json:"sub"`
	Username       string   `json:"username"`
	Email          string   `json:"email"`
	Role           string   `json:"role"`
	SlugPrefix     string   `json:"slugPrefix"`
	AllowedDomains []string `json:"allowedDomains,omitempty"`
	Status         string   `json:"status"`
	Permissions    []string `json:"permissions"`
}

func (h *MeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "unauthorized"}, http.StatusUnauthorized)
		return
	}

	var perms []string
	if h.permSvc != nil {
		perms, _ = h.permSvc.GetUserPermissions(r.Context(), user.Sub)
	}
	if perms == nil {
		perms = []string{}
	}

	var allowedDomains []string
	if user.AllowedDomains != "" {
		_ = json.Unmarshal([]byte(user.AllowedDomains), &allowedDomains)
	}

	writeJSON(w, meResponse{
		ID:             user.ID.String(),
		Sub:            user.Sub,
		Username:       user.Username,
		Email:          user.Email,
		Role:           user.Role,
		SlugPrefix:     user.SlugPrefix,
		AllowedDomains: allowedDomains,
		Status:         string(user.Status),
		Permissions:    perms,
	}, http.StatusOK)
}

