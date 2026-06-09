package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unified-backend/internal/controller"
	"unified-backend/internal/domain"
	"unified-backend/internal/middleware"
	"unified-backend/internal/repository/postgres"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type PermInvalidator interface {
	InvalidateUser(userID uuid.UUID)
}

type UserHandler struct {
	userRepo  *postgres.UserRepository
	auditRepo *postgres.AuditRepository
	permCtrl  controller.PermChecker
	inv       PermInvalidator
}

func NewUserHandler(
	userRepo *postgres.UserRepository,
	auditRepo *postgres.AuditRepository,
	permCtrl controller.PermChecker,
	inv PermInvalidator,
) *UserHandler {
	return &UserHandler{
		userRepo:  userRepo,
		auditRepo: auditRepo,
		permCtrl:  permCtrl,
		inv:       inv,
	}
}

func (h *UserHandler) recordAuditAsync(entry *domain.AuditEntry) {
	if h.auditRepo == nil || entry == nil {
		return
	}
	go func(e *domain.AuditEntry) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		h.auditRepo.Record(ctx, e)
	}(entry)
}

func (h *UserHandler) requirePerm(w http.ResponseWriter, r *http.Request, action string) bool {
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

func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	if !h.requirePerm(w, r, domain.PermUsersView) {
		return
	}
	users, err := h.userRepo.ListAll(r.Context())
	if err != nil {
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}
	writeJSON(w, users, http.StatusOK)
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	if !h.requirePerm(w, r, domain.PermUsersView) {
		return
	}
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

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	if !h.requirePerm(w, r, domain.PermUsersManage) {
		return
	}
	sub := chi.URLParam(r, "sub")
	if sub == "" {
		writeJSON(w, map[string]string{"error": "sub required"}, http.StatusBadRequest)
		return
	}

	bodyBytes, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	var p struct {
		Status         *string   `json:"status"`
		SlugPrefix     *string   `json:"slugPrefix"`
		AllowedDomains *[]string `json:"allowedDomains"`
	}
	if err := json.Unmarshal(bodyBytes, &p); err != nil {
		writeJSON(w, map[string]string{"error": "invalid json"}, http.StatusBadRequest)
		return
	}

	fields := map[string]any{}
	if p.Status != nil {
		fields["status"] = strings.TrimSpace(*p.Status)
	}
	if p.SlugPrefix != nil {
		fields["slug_prefix"] = strings.TrimSpace(*p.SlugPrefix)
	}
	if p.AllowedDomains != nil {
		cleaned := make([]string, 0, len(*p.AllowedDomains))
		for _, d := range *p.AllowedDomains {
			d = strings.TrimSpace(d)
			if d != "" {
				cleaned = append(cleaned, d)
			}
		}
		encoded, _ := json.Marshal(cleaned)
		fields["allowed_domains"] = string(encoded)
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
	h.recordAuditAsync(&domain.AuditEntry{
		UserSub:  actor.Sub,
		Username: actor.Username,
		Action:   "user.update",
		Resource: sub,
		Result:   "success",
		Details:  map[string]any{"body": string(bodyBytes)},
	})

	updated, _ := h.userRepo.GetBySub(r.Context(), sub)
	writeJSON(w, updated, http.StatusOK)
}

func (h *UserHandler) PatchUserRole(w http.ResponseWriter, r *http.Request) {
	if !h.requirePerm(w, r, domain.PermUsersManage) {
		return
	}
	sub := chi.URLParam(r, "sub")
	if sub == "" {
		writeJSON(w, map[string]string{"error": "sub required"}, http.StatusBadRequest)
		return
	}

	var p struct {
		RoleID string `json:"roleId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil || p.RoleID == "" {
		writeJSON(w, map[string]string{"error": "roleId required"}, http.StatusBadRequest)
		return
	}
	parsedID, err := uuid.Parse(p.RoleID)
	if err != nil {
		writeJSON(w, map[string]string{"error": "invalid roleId"}, http.StatusBadRequest)
		return
	}

	var roleName string
	if err := h.userRepo.Pool().QueryRow(
		r.Context(),
		`SELECT name FROM roles WHERE id = $1`, parsedID,
	).Scan(&roleName); err != nil {
		writeJSON(w, map[string]string{"error": "role not found"}, http.StatusNotFound)
		return
	}

	if err := h.userRepo.UpdateBySubFields(r.Context(), sub, map[string]any{
		"role_id":   parsedID,
		"role_text": roleName,
	}); err != nil {
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}

	if h.inv != nil {
		targetUser, _ := h.userRepo.GetBySub(r.Context(), sub)
		if targetUser != nil {
			h.inv.InvalidateUser(targetUser.ID)
		}
	}

	actor := middleware.UserFromCtx(r.Context())
	h.recordAuditAsync(&domain.AuditEntry{
		UserSub:  actor.Sub,
		Username: actor.Username,
		Action:   "user.role.assign",
		Resource: sub,
		Result:   "success",
		Details:  map[string]any{"roleId": p.RoleID, "roleName": roleName},
	})

	writeJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
}

func (h *UserHandler) GetUserLinks(w http.ResponseWriter, r *http.Request) {
	if !h.requirePerm(w, r, domain.PermUsersView) {
		return
	}
	sub := chi.URLParam(r, "sub")
	if sub == "" {
		writeJSON(w, map[string]string{"error": "sub required"}, http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]any{
		"items":   []any{},
		"message": "use GET /api/shlink/short-urls with user context",
	}, http.StatusOK)
}

func (h *UserHandler) ListAudit(w http.ResponseWriter, r *http.Request) {
	if !h.requirePerm(w, r, domain.PermUsersView) {
		return
	}
	page, perPage := 1, 20
	if v := r.URL.Query().Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			page = n
		}
	}
	if v := r.URL.Query().Get("perPage"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > 100 {
				n = 100
			}
			perPage = n
		}
	}
	result, err := h.auditRepo.List(r.Context(), postgres.AuditFilter{Page: page, Limit: perPage})
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

