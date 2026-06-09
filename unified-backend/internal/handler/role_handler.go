package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"unified-backend/internal/controller"
	"unified-backend/internal/domain"
	"unified-backend/internal/middleware"
	"unified-backend/internal/repository/postgres"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type RoleInvalidator interface {
	InvalidateRole(roleName string)
}

type RoleHandler struct {
	roleRepo *postgres.RoleRepository
	permRepo *postgres.PermissionRepository
	permCtrl controller.PermChecker
	inv      RoleInvalidator
}

func NewRoleHandler(
	roleRepo *postgres.RoleRepository,
	permRepo *postgres.PermissionRepository,
	permCtrl controller.PermChecker,
	inv RoleInvalidator,
) *RoleHandler {
	return &RoleHandler{
		roleRepo: roleRepo,
		permRepo: permRepo,
		permCtrl: permCtrl,
		inv:      inv,
	}
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

func (h *RoleHandler) GetRole(w http.ResponseWriter, r *http.Request) {
	if !h.requirePerm(w, r, domain.PermRolesView) {
		return
	}
	roleIDStr := chi.URLParam(r, "role")
	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		writeJSON(w, map[string]string{"error": "invalid role id"}, http.StatusBadRequest)
		return
	}
	role, err := h.roleRepo.GetByID(r.Context(), roleID)
	if err != nil || role == nil {
		writeJSON(w, map[string]string{"error": "not found"}, http.StatusNotFound)
		return
	}
	perms, _ := h.roleRepo.GetPermissions(r.Context(), roleID)
	role.Permissions = perms
	writeJSON(w, role, http.StatusOK)
}

func (h *RoleHandler) UpsertRolePermissions(w http.ResponseWriter, r *http.Request) {
	if !h.requirePerm(w, r, domain.PermRolesManage) {
		return
	}
	roleIDStr := chi.URLParam(r, "role")
	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		writeJSON(w, map[string]string{"error": "invalid role id"}, http.StatusBadRequest)
		return
	}
	role, err := h.roleRepo.GetByID(r.Context(), roleID)
	if err != nil || role == nil {
		writeJSON(w, map[string]string{"error": "role not found"}, http.StatusNotFound)
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
	if h.inv != nil {
		h.inv.InvalidateRole(role.Name)
	}
	writeJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
}

func (h *RoleHandler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	if !h.requirePerm(w, r, domain.PermRolesManage) {
		return
	}
	roleIDStr := chi.URLParam(r, "role")
	roleID, err := uuid.Parse(roleIDStr)
	if err != nil {
		writeJSON(w, map[string]string{"error": "invalid role id"}, http.StatusBadRequest)
		return
	}
	role, err := h.roleRepo.GetByID(r.Context(), roleID)
	if err != nil || role == nil {
		writeJSON(w, map[string]string{"error": "not found"}, http.StatusNotFound)
		return
	}
	if role.Name == "admin" || role.Name == "viewer" {
		writeJSON(w, map[string]string{"error": "cannot delete system role"}, http.StatusForbidden)
		return
	}
	// Сначала удаляем permissions
	if err := h.roleRepo.SetPermissions(r.Context(), roleID, []uuid.UUID{}); err != nil {
		slog.Error("role_handler: clear permissions failed", "err", err)
	}
	// Затем удаляем роль (каскад удалит из role_permissions_v2)
	_, err = h.roleRepo.Pool().Exec(r.Context(), `DELETE FROM roles WHERE id = $1`, roleID)
	if err != nil {
		slog.Error("role_handler: delete role failed", "err", err)
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}
	if h.inv != nil {
		h.inv.InvalidateRole(role.Name)
	}
	writeJSON(w, map[string]string{"status": "deleted"}, http.StatusOK)
}

