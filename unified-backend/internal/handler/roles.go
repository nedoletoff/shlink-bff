package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"unified-backend/internal/config"
	"unified-backend/internal/domain"
	"unified-backend/internal/repository/postgres"
	"unified-backend/internal/service"
)

// RolesHandler — управление permissions ролей.
// Доступен только ролям с can_manage_roles = true.
type RolesHandler struct {
	cache     *service.PermissionsCache
	permsRepo *postgres.RolePermissionsRepository
	cfg       *config.Config
}

func NewRolesHandler(cache *service.PermissionsCache, repo *postgres.RolePermissionsRepository, cfg *config.Config) *RolesHandler {
	return &RolesHandler{cache: cache, permsRepo: repo, cfg: cfg}
}

type roleEntry struct {
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
	UsersCount  int      `json:"usersCount"`
}

type roleMapping struct {
	KcGroup string `json:"kcGroup"`
	AppRole string `json:"appRole"`
}

type rolesListResponse struct {
	Roles    []roleEntry   `json:"roles"`
	Mappings []roleMapping `json:"mappings"`
}

// GET /api/admin/roles — список всех ролей с их permissions + маппинги из конфига
func (h *RolesHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	perms := h.cache.GetAll()

	roles := make([]roleEntry, 0, len(perms))
	for _, p := range perms {
		flags := permToStringSlice(p)
		roles = append(roles, roleEntry{
			Role:        p.Role,
			Permissions: flags,
			UsersCount:  0, // не считаем здесь — дорого; при необходимости добавить userRepo
		})
	}

	// Маппинги из ROLE_GROUPS: keycloak-group → app-role
	mappings := make([]roleMapping, 0, len(h.cfg.RoleGroups))
	for group, role := range h.cfg.RoleGroups {
		mappings = append(mappings, roleMapping{KcGroup: group, AppRole: role})
	}

	writeJSON(w, rolesListResponse{Roles: roles, Mappings: mappings}, http.StatusOK)
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

// permToStringSlice переводит флаги RolePermissions в список строк (только true-флаги).
func permToStringSlice(p domain.RolePermissions) []string {
	var out []string
	if p.CanViewOwnLinks          { out = append(out, "canViewOwnLinks") }
	if p.CanViewAllLinks          { out = append(out, "canViewAllLinks") }
	if p.CanCreateLinks           { out = append(out, "canCreateLinks") }
	if p.CanCreateWithCustomSlug  { out = append(out, "canCreateWithCustomSlug") }
	if p.CanCreateWithoutSlug     { out = append(out, "canCreateWithoutSlug") }
	if p.CanEditOwnLinks          { out = append(out, "canEditOwnLinks") }
	if p.CanEditAllLinks          { out = append(out, "canEditAllLinks") }
	if p.CanDeleteOwnLinks        { out = append(out, "canDeleteOwnLinks") }
	if p.CanDeleteAllLinks        { out = append(out, "canDeleteAllLinks") }
	if p.CanManageOwnTags         { out = append(out, "canManageOwnTags") }
	if p.CanManageAllTags         { out = append(out, "canManageAllTags") }
	if p.CanViewOwnStats          { out = append(out, "canViewOwnStats") }
	if p.CanViewAllStats          { out = append(out, "canViewAllStats") }
	if p.CanViewAuditLogs         { out = append(out, "canViewAuditLogs") }
	if p.CanManageUsers           { out = append(out, "canManageUsers") }
	if p.CanManageRoles           { out = append(out, "canManageRoles") }
	return out
}
