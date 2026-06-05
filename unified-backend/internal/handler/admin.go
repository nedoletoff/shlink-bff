package handler

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"unified-backend/internal/domain"
	"unified-backend/internal/middleware"
	"unified-backend/internal/repository/postgres"
)

type AdminHandler struct {
	userRepo  *postgres.UserRepository
	auditRepo *postgres.AuditRepository
	rolesRepo *postgres.RolePermissionsRepository
}

func NewAdminHandler(
	userRepo *postgres.UserRepository,
	auditRepo *postgres.AuditRepository,
	rolesRepo *postgres.RolePermissionsRepository,
) *AdminHandler {
	return &AdminHandler{userRepo: userRepo, auditRepo: auditRepo, rolesRepo: rolesRepo}
}

// recordAuditAsync — записи аудита в горутине с детачнутым контекстом.
func (h *AdminHandler) recordAuditAsync(entry *domain.AuditEntry) {
	if h.auditRepo == nil || entry == nil {
		return
	}
	go func(e *domain.AuditEntry) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		h.auditRepo.Record(ctx, e)
	}(entry)
}

// GET /api/admin/users
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.userRepo.ListAll(r.Context())
	if err != nil {
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}
	writeJSON(w, users, http.StatusOK)
}

// GET /api/admin/users/{sub}
func (h *AdminHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "sub")
	if sub == "" {
		writeJSON(w, map[string]string{"error": "sub required"}, http.StatusBadRequest)
		return
	}
	user, err := h.userRepo.GetBySub(r.Context(), sub)
	if err != nil || user == nil {
		writeJSON(w, map[string]string{"error": "not found"}, http.StatusNotFound)
		return
	}
	writeJSON(w, user, http.StatusOK)
}

// GET /api/admin/users/{sub}/links
func (h *AdminHandler) GetUserLinks(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "sub")
	if sub == "" {
		writeJSON(w, map[string]string{"error": "sub required"}, http.StatusBadRequest)
		return
	}
	if _, err := h.userRepo.GetBySub(r.Context(), sub); err != nil {
		writeJSON(w, map[string]string{"error": "not found"}, http.StatusNotFound)
		return
	}
	writeJSON(w, map[string]any{
		"items":   []any{},
		"message": "use /api/shlink/short-urls with user context or add explicit aggregation later",
	}, http.StatusOK)
}

// GET /api/admin/logs
func (h *AdminHandler) ListLogs(w http.ResponseWriter, r *http.Request) {
	page := 1
	perPage := 20
	q := r.URL.Query()
	if v := q.Get("page"); v != "" {
		if n, err := parsePositiveInt(v); err == nil {
			page = n
		}
	}
	if v := q.Get("perPage"); v != "" {
		if n, err := parsePositiveInt(v); err == nil {
			if n > 100 {
				n = 100
			}
			perPage = n
		}
	}

	result, err := h.auditRepo.List(r.Context(), postgres.AuditFilter{
		Page:  page,
		Limit: perPage,
	})
	if err != nil {
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{
		"items":   result.Logs,
		"page":    page,
		"perPage": perPage,
		"total":   result.Total,
	}, http.StatusOK)
}

// GET /api/admin/roles
func (h *AdminHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	rows, err := h.rolesRepo.GetAll(r.Context())
	if err != nil {
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}
	writeJSON(w, rows, http.StatusOK)
}

// GET /api/admin/roles/{role}
func (h *AdminHandler) GetRole(w http.ResponseWriter, r *http.Request) {
	role := chi.URLParam(r, "role")
	if strings.TrimSpace(role) == "" {
		writeJSON(w, map[string]string{"error": "role required"}, http.StatusBadRequest)
		return
	}
	row, err := h.rolesRepo.GetByRole(r.Context(), role)
	if err != nil || row == nil {
		writeJSON(w, map[string]string{"error": "not found"}, http.StatusNotFound)
		return
	}
	writeJSON(w, row, http.StatusOK)
}

// parsePositiveInt — небольшой локальный helper
func parsePositiveInt(s string) (int, error) {
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, io.ErrUnexpectedEOF
		}
		n = n*10 + int(ch-'0')
	}
	if n <= 0 {
		return 0, io.ErrUnexpectedEOF
	}
	return n, nil
}

type updateUserPayload struct {
	Role       *string `json:"role"`
	Status     *string `json:"status"`
	SlugPrefix *string `json:"slugPrefix"`
}

// PUT /api/admin/users/{sub}
func (h *AdminHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "sub")
	if sub == "" {
		writeJSON(w, map[string]string{"error": "sub required"}, http.StatusBadRequest)
		return
	}
	if _, err := h.userRepo.GetBySub(r.Context(), sub); err != nil {
		writeJSON(w, map[string]string{"error": "not found"}, http.StatusNotFound)
		return
	}

	bodyBytes, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	var p updateUserPayload
	if err := json.Unmarshal(bodyBytes, &p); err != nil {
		writeJSON(w, map[string]string{"error": "invalid json"}, http.StatusBadRequest)
		return
	}

	fields := map[string]any{}
	if p.Role != nil {
		fields["role"] = strings.TrimSpace(*p.Role)
	}
	if p.Status != nil {
		fields["status"] = strings.TrimSpace(*p.Status)
	}
	if p.SlugPrefix != nil {
		fields["slug_prefix"] = strings.TrimSpace(*p.SlugPrefix)
	}
	if len(fields) == 0 {
		writeJSON(w, map[string]string{"error": "no fields to update"}, http.StatusBadRequest)
		return
	}

	if err := h.userRepo.UpdateBySubFields(r.Context(), sub, fields); err != nil {
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}

	actor := middleware.UserFromCtx(r.Context())
	actorSub, actorRole, actorUsername := "", "", ""
	if actor != nil {
		actorSub = actor.Sub
		actorRole = string(actor.Role)
		actorUsername = actor.Username
	}
	h.recordAuditAsync(&domain.AuditEntry{
		UserSub:  actorSub,
		Username: actorUsername,
		Role:     actorRole,
		Action:   "admin.user.update",
		Resource: sub,
		Result:   "success",
		Details:  map[string]any{"body": string(bodyBytes)},
	})

	updated, _ := h.userRepo.GetBySub(r.Context(), sub)
	if updated == nil {
		writeJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
		return
	}
	writeJSON(w, updated, http.StatusOK)
}

type apiKeyPayload struct {
	APIKey string `json:"apiKey"`
}

// PUT /api/admin/users/{sub}/apikey
func (h *AdminHandler) UpdateAPIKey(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "sub")
	if sub == "" {
		writeJSON(w, map[string]string{"error": "sub required"}, http.StatusBadRequest)
		return
	}
	if _, err := h.userRepo.GetBySub(r.Context(), sub); err != nil {
		writeJSON(w, map[string]string{"error": "not found"}, http.StatusNotFound)
		return
	}

	bodyBytes, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	var p apiKeyPayload
	if err := json.Unmarshal(bodyBytes, &p); err != nil {
		writeJSON(w, map[string]string{"error": "invalid json"}, http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(p.APIKey) == "" {
		writeJSON(w, map[string]string{"error": "apiKey required"}, http.StatusBadRequest)
		return
	}

	if err := h.userRepo.UpdateBySubFields(r.Context(), sub, map[string]any{"shlink_api_key": p.APIKey}); err != nil {
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}

	actor := middleware.UserFromCtx(r.Context())
	actorSub2, actorRole2, actorUsername2 := "", "", ""
	if actor != nil {
		actorSub2 = actor.Sub
		actorRole2 = string(actor.Role)
		actorUsername2 = actor.Username
	}
	h.recordAuditAsync(&domain.AuditEntry{
		UserSub:  actorSub2,
		Username: actorUsername2,
		Role:     actorRole2,
		Action:   "admin.user.apikey.update",
		Resource: sub,
		Result:   "success",
	})

	writeJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
}

type prefixPayload struct {
	SlugPrefix string `json:"slugPrefix"`
}

// PUT /api/admin/users/{sub}/prefix
func (h *AdminHandler) UpdateSlugPrefix(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "sub")
	if sub == "" {
		writeJSON(w, map[string]string{"error": "sub required"}, http.StatusBadRequest)
		return
	}
	if _, err := h.userRepo.GetBySub(r.Context(), sub); err != nil {
		writeJSON(w, map[string]string{"error": "not found"}, http.StatusNotFound)
		return
	}

	bodyBytes, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	var p prefixPayload
	if err := json.Unmarshal(bodyBytes, &p); err != nil {
		writeJSON(w, map[string]string{"error": "invalid json"}, http.StatusBadRequest)
		return
	}

	if err := h.userRepo.UpdateBySubFields(r.Context(), sub, map[string]any{"slug_prefix": strings.TrimSpace(p.SlugPrefix)}); err != nil {
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}

	actor := middleware.UserFromCtx(r.Context())
	actorSub3, actorRole3, actorUsername3 := "", "", ""
	if actor != nil {
		actorSub3 = actor.Sub
		actorRole3 = string(actor.Role)
		actorUsername3 = actor.Username
	}
	h.recordAuditAsync(&domain.AuditEntry{
		UserSub:  actorSub3,
		Username: actorUsername3,
		Role:     actorRole3,
		Action:   "admin.user.prefix.update",
		Resource: sub,
		Result:   "success",
		Details:  map[string]any{"body": string(bodyBytes)},
	})

	writeJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
}

type rolePermissionsPayload struct {
	Permissions []string `json:"permissions"`
}

// PUT /api/admin/roles/{role}/permissions
func (h *AdminHandler) UpdateRolePermissions(w http.ResponseWriter, r *http.Request) {
	role := chi.URLParam(r, "role")
	if strings.TrimSpace(role) == "" {
		writeJSON(w, map[string]string{"error": "role required"}, http.StatusBadRequest)
		return
	}

	body, _ := io.ReadAll(r.Body)

	var p domain.RolePermissions
	if err := json.Unmarshal(body, &p); err != nil {
		writeJSON(w, map[string]string{"error": "invalid json"}, http.StatusBadRequest)
		return
	}
	p.Role = role

	if err := h.rolesRepo.Upsert(r.Context(), &p); err != nil {
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}

	actor := middleware.UserFromCtx(r.Context())
	actorSub5, actorRole5, actorUsername5 := "", "", ""
	if actor != nil {
		actorSub5 = actor.Sub
		actorRole5 = string(actor.Role)
		actorUsername5 = actor.Username
	}
	h.recordAuditAsync(&domain.AuditEntry{
		UserSub:  actorSub5,
		Username: actorUsername5,
		Role:     actorRole5,
		Action:   "admin.role.permissions.update",
		Result:   "success",
		Details:  map[string]any{"body": string(body)},
	})

	writeJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
}

// GET /api/admin/settings
func (h *AdminHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"oidcEnabled":        true,
		"proxyAuthEnabled":   true,
		"shlinkIntegration":  true,
		"auditEnabled":       h.auditRepo != nil,
	}, http.StatusOK)
}

// PATCH /api/admin/settings
func (h *AdminHandler) PatchSettings(w http.ResponseWriter, r *http.Request) {
	bodyBytes, _ := io.ReadAll(r.Body)
	actor := middleware.UserFromCtx(r.Context())
	actorSub4, actorRole4, actorUsername4 := "", "", ""
	if actor != nil {
		actorSub4 = actor.Sub
		actorRole4 = string(actor.Role)
		actorUsername4 = actor.Username
	}
	h.recordAuditAsync(&domain.AuditEntry{
		UserSub:  actorSub4,
		Username: actorUsername4,
		Role:     actorRole4,
		Action:   "admin.settings.patch",
		Result:   "success",
		Details:  map[string]any{"body": string(bodyBytes)},
	})
	writeJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
}

// stubbed slog reference to avoid unused import if logging removed later
var _ = slog.Info
