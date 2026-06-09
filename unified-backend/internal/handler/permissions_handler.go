package handler

import (
	"net/http"

	"unified-backend/internal/domain"
)

// PermissionsHandler — GET /api/permissions
// Возвращает полный список всех возможных разрешений системы.
// Фронтенд использует для динамического построения UI управления ролями.
type PermissionsHandler struct{}

func NewPermissionsHandler() *PermissionsHandler {
	return &PermissionsHandler{}
}

type permissionsResponse struct {
	Permissions []string `json:"permissions"`
}

// ListPermissions — GET /api/permissions
func (h *PermissionsHandler) ListPermissions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, permissionsResponse{
		Permissions: domain.AllPermissions,
	}, http.StatusOK)
}
