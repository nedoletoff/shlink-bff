package handler

import (
	"context"
	"encoding/json"
	"io"
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
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}
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
