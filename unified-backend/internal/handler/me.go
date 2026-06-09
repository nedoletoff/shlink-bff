package handler

import (
	"net/http"

	"unified-backend/internal/middleware"
	"unified-backend/internal/service"
)

// PermissionServiceIface — минимальный интерфейс для MeHandler.
type PermissionServiceIface interface {
	GetUserPermissions(ctx interface {
		Deadline() (interface{}, bool)
		Done() <-chan struct{}
		Err() error
		Value(interface{}) interface{}
	}, userIDStr string) ([]string, error)
}

// MeHandler — GET /api/me
type MeHandler struct {
	userRepo UserRepo
	permSvc  *service.PermissionService
}

func NewMeHandler(userRepo UserRepo, permSvc *service.PermissionService) *MeHandler {
	return &MeHandler{userRepo: userRepo, permSvc: permSvc}
}

type meResponse struct {
	ID          string   `json:"id"`
	Sub         string   `json:"sub"`
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	Role        string   `json:"role"`
	SlugPrefix  string   `json:"slugPrefix"`
	Status      string   `json:"status"`
	Permissions []string `json:"permissions"`
}

func (h *MeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "unauthorized"}, http.StatusUnauthorized)
		return
	}

	// Используем унифицированный сервис (tот же метод, что в user_handler.go)
	var perms []string
	if h.permSvc != nil {
		// GetUserPermissions принимает Sub пользователя
		perms, _ = h.permSvc.GetUserPermissions(r.Context(), user.Sub)
	}
	if perms == nil {
		perms = []string{}
	}

	writeJSON(w, meResponse{
		ID:          user.ID.String(),
		Sub:         user.Sub,
		Username:    user.Username,
		Email:       user.Email,
		Role:        user.Role,
		SlugPrefix:  user.SlugPrefix,
		Status:      string(user.Status),
		Permissions: perms,
	}, http.StatusOK)
}
