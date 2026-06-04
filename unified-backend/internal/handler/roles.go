package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"unified-backend/internal/domain"
	"unified-backend/internal/repository/postgres"
	"unified-backend/internal/service"
)

// RolesHandler — управление permissions ролей.
// Доступен только ролям с can_manage_roles = true.
type RolesHandler struct {
	cache     *service.PermissionsCache
	permsRepo *postgres.RolePermissionsRepository
}

func NewRolesHandler(cache *service.PermissionsCache, repo *postgres.RolePermissionsRepository) *RolesHandler {
	return &RolesHandler{cache: cache, permsRepo: repo}
}

// GET /api/admin/roles — список всех ролей с их permissions
func (h *RolesHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, h.cache.GetAll(), http.StatusOK)
}

// GET /api/admin/roles/{role} — permissions конкретной роли
func (h *RolesHandler) GetRole(w http.ResponseWriter, r *http.Request) {
	role := chi.URLParam(r, "role")
	p := h.cache.Get(role)
	if p.Role == "" {
		p.Role = role
	}
	writeJSON(w, p, http.StatusOK)
}

// PUT /api/admin/roles/{role}/permissions — полное обновление permissions
func (h *RolesHandler) UpsertRolePermissions(w http.ResponseWriter, r *http.Request) {
	role := chi.URLParam(r, "role")
	if role == "" {
		writeJSON(w, map[string]string{"error": "role required"}, http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
	if err != nil {
		writeJSON(w, map[string]string{"error": "bad request"}, http.StatusBadRequest)
		return
	}

	var p domain.RolePermissions
	if err := json.Unmarshal(body, &p); err != nil {
		writeJSON(w, map[string]string{"error": "invalid json"}, http.StatusBadRequest)
		return
	}
	p.Role = role // role из URL — канонический источник

	if err := h.permsRepo.Upsert(r.Context(), &p); err != nil {
		slog.Error("roles: upsert permissions failed", "role", role, "err", err)
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}

	// Обновляем кеш сразу — без перезапуска сервиса
	h.cache.Set(p)
	slog.Info("roles: permissions updated", "role", role)
	writeJSON(w, p, http.StatusOK)
}
