package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"unified-backend/internal/controller"
	"unified-backend/internal/domain"
	"unified-backend/internal/middleware"
	"unified-backend/internal/repository/postgres"
)

// RoleHandler — управление ролями через RBAC.
// Заменяет RolesHandler.
type RoleHandler struct {
	roleRepo   *postgres.RoleRepository
	permRepo   *postgres.PermissionRepository
	permCtrl   controller.PermChecker
}

func NewRoleHandler(
	roleRepo *postgres.RoleRepository,
	permRepo *postgres.PermissionRepository,
	permCtrl controller.PermChecker,
) *RoleHandler {
	return &RoleHandler{roleRepo: roleRepo, permRepo: permRepo, permCtrl: permCtrl}
}

func (h *RoleHandler) requirePerm(w http.ResponseWriter, r *http.Request, action string) bool {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "unauthorized"}, http.StatusUnauthorized)
		return false
	}
	ok, err := h.permCtrl.Check(r.Context(), user.ID, action)
	if err != nil {
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return false
	}
	if !ok {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return false
	}
	return true
}

// GET /api/roles
func (h *RoleHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	if !h.requirePerm(w, r, domain.PermRolesView) {
		return
	}
	roles, err := h.roleRepo.GetAll(r.Context())
	if err != nil {
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}
	writeJSON(w, roles, http.StatusOK)
}

// POST /api/roles
func (h *RoleHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	if !h.requirePerm(w, r, domain.PermRolesManage) {
		return
	}

	var p struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
	if err != nil {
		writeJSON(w, map[string]string{"error": "bad request"}, http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(body, &p); err != nil || strings.TrimSpace(p.Name) == "" {
		writeJSON(w, map[string]string{"error": "name required"}, http.StatusBadRequest)
		return
	}

	role, err := h.roleRepo.Create(r.Context(), strings.TrimSpace(p.Name), p.Description)
	if err != nil {
		slog.Error("role_handler: create failed", "err", err)
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}
	writeJSON(w, role, http.StatusCreated)
}

// GET /api/roles/{id}/permissions
func (h *RoleHandler) GetRolePermissions(w http.ResponseWriter, r *http.Request) {
	if !h.requirePerm(w, r, domain.PermRolesView) {
		return
	}
	roleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, map[string]string{"error": "invalid role id"}, http.StatusBadRequest)
		return
	}
	perms, err := h.roleRepo.GetPermissions(r.Context(), roleID)
	if err != nil {
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}
	writeJSON(w, perms, http.StatusOK)
}

// PUT /api/roles/{id}/permissions — полная замена набора разрешений.
func (h *RoleHandler) SetRolePermissions(w http.ResponseWriter, r *http.Request) {
	if !h.requirePerm(w, r, domain.PermRolesManage) {
		return
	}
	roleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, map[string]string{"error": "invalid role id"}, http.StatusBadRequest)
		return
	}

	var payload struct {
		PermissionIDs []string `json:"permissionIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, map[string]string{"error": "invalid json"}, http.StatusBadRequest)
		return
	}

	permIDs := make([]uuid.UUID, 0, len(payload.PermissionIDs))
	for _, s := range payload.PermissionIDs {
		id, err := uuid.Parse(s)
		if err != nil {
			writeJSON(w, map[string]string{"error": "invalid permissionId: " + s}, http.StatusBadRequest)
			return
		}
		permIDs = append(permIDs, id)
	}

	if err := h.roleRepo.SetPermissions(r.Context(), roleID, permIDs); err != nil {
		slog.Error("role_handler: set permissions failed", "err", err)
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
}
