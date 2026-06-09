package handler

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"unified-backend/internal/config"
	"unified-backend/internal/domain"
	"unified-backend/internal/middleware"
	"unified-backend/internal/service"
)

// RolesRepository — минимальный интерфейс для персистентности permissions ролей.
type RolesRepository interface {
	GetAll(ctx context.Context) ([]domain.RolePermissions, error)
	Upsert(ctx context.Context, p *domain.RolePermissions) error
	Delete(ctx context.Context, role string) error
}

// RolesHandler — управление permissions ролей.
type RolesHandler struct {
	cache     service.PermissionsCacheAdmin
	permsRepo RolesRepository
	cfg       *config.Config
}

func NewRolesHandler(cache service.PermissionsCacheAdmin, repo RolesRepository, cfg *config.Config) *RolesHandler {
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

func (h *RolesHandler) requireManageRoles(w http.ResponseWriter, r *http.Request) *domain.User {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return nil
	}
	roles := middleware.RolesFromCtx(r.Context(), string(user.Role))
	if !h.cache.GetMerged(roles).CanManageRoles {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return nil
	}
	return user
}

// GET /api/roles
func (h *RolesHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	if h.requireManageRoles(w, r) == nil {
		return
	}
	perms := h.cache.GetAll()

	roles := make([]roleEntry, 0, len(perms))
	for _, p := range perms {
		flags := permToStringSlice(p)
		roles = append(roles, roleEntry{
			Role:        p.Role,
			Permissions: flags,
			UsersCount:  0,
		})
	}

	mappings := make([]roleMapping, 0, len(h.cfg.RoleGroups))
	for group, role := range h.cfg.RoleGroups {
		mappings = append(mappings, roleMapping{KcGroup: group, AppRole: role})
	}

	writeJSON(w, rolesListResponse{Roles: roles, Mappings: mappings}, http.StatusOK)
}

// GET /api/roles/{role}
func (h *RolesHandler) GetRole(w http.ResponseWriter, r *http.Request) {
	if h.requireManageRoles(w, r) == nil {
		return
	}
	role := chi.URLParam(r, "role")
	p := h.cache.Get(role)
	if p.Role == "" {
		p.Role = role
	}
	writeJSON(w, p, http.StatusOK)
}

// POST /api/roles
func (h *RolesHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	if h.requireManageRoles(w, r) == nil {
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

	if strings.TrimSpace(p.Role) == "" {
		writeJSON(w, map[string]string{"error": "role name required"}, http.StatusBadRequest)
		return
	}

	if err := h.permsRepo.Upsert(r.Context(), &p); err != nil {
		slog.Error("roles: create failed", "role", p.Role, "err", err)
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}

	h.cache.Reload(r.Context())
	writeJSON(w, p, http.StatusCreated)
}

// PUT /api/roles/{role}/permissions
func (h *RolesHandler) UpsertRolePermissions(w http.ResponseWriter, r *http.Request) {
	if h.requireManageRoles(w, r) == nil {
		return
	}

	role := chi.URLParam(r, "role")
	if strings.TrimSpace(role) == "" {
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
	p.Role = role

	if err := h.permsRepo.Upsert(r.Context(), &p); err != nil {
		slog.Error("roles: upsert permissions failed", "role", role, "err", err)
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}

	h.cache.Reload(r.Context())
	writeJSON(w, p, http.StatusOK)
}

// DELETE /api/roles/{role}
func (h *RolesHandler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	if h.requireManageRoles(w, r) == nil {
		return
	}

	role := chi.URLParam(r, "role")
	if strings.TrimSpace(role) == "" {
		writeJSON(w, map[string]string{"error": "role required"}, http.StatusBadRequest)
		return
	}

	if err := h.permsRepo.Delete(r.Context(), role); err != nil {
		slog.Error("roles: delete failed", "role", role, "err", err)
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}

	h.cache.Reload(r.Context())
	w.WriteHeader(http.StatusNoContent)
}

// permToStringSlice переводит флаги RolePermissions в список строк.
// Порядок соответствует группировке в domain.RolePermissions.
func permToStringSlice(p domain.RolePermissions) []string {
	var out []string
	// Просмотр ссылок
	if p.CanViewOwnLinks          { out = append(out, "canViewOwnLinks") }
	if p.CanViewAllLinks          { out = append(out, "canViewAllLinks") }
	// Создание
	if p.CanCreateLinks           { out = append(out, "canCreateLinks") }
	if p.CanCreateWithCustomSlug  { out = append(out, "canCreateWithCustomSlug") }
	if p.CanCreateWithoutSlug     { out = append(out, "canCreateWithoutSlug") }
	// Редактирование
	if p.CanEditOwnLinks          { out = append(out, "canEditOwnLinks") }
	if p.CanEditAllLinks          { out = append(out, "canEditAllLinks") }
	// Удаление (soft)
	if p.CanDeleteOwnLinks        { out = append(out, "canDeleteOwnLinks") }
	if p.CanDeleteAllLinks        { out = append(out, "canDeleteAllLinks") }
	// Деактивация / реактивация
	if p.CanDeactivateOwnLinks    { out = append(out, "canDeactivateOwnLinks") }
	if p.CanDeactivateAllLinks    { out = append(out, "canDeactivateAllLinks") }
	if p.CanReactivateOwnLinks    { out = append(out, "canReactivateOwnLinks") }
	if p.CanReactivateAllLinks    { out = append(out, "canReactivateAllLinks") }
	// Permanent delete
	if p.CanDeleteOwnLinksPermanently { out = append(out, "canDeleteOwnLinksPermanently") }
	if p.CanDeleteAllLinksPermanently { out = append(out, "canDeleteAllLinksPermanently") }
	// Теги
	if p.CanManageOwnTags         { out = append(out, "canManageOwnTags") }
	if p.CanManageAllTags         { out = append(out, "canManageAllTags") }
	// Статистика
	if p.CanViewOwnStats          { out = append(out, "canViewOwnStats") }
	if p.CanViewAllStats          { out = append(out, "canViewAllStats") }
	// Управление
	if p.CanViewAuditLogs         { out = append(out, "canViewAuditLogs") }
	if p.CanManageUsers           { out = append(out, "canManageUsers") }
	if p.CanManageRoles           { out = append(out, "canManageRoles") }
	if p.CanManageSettings        { out = append(out, "canManageSettings") }
	return out
}
