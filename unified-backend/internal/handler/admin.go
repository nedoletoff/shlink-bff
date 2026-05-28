package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"unified-backend/internal/domain"
	"unified-backend/internal/middleware"
	"unified-backend/internal/repository/postgres"
)

type AdminHandler struct {
	userRepo  *postgres.UserRepository
	auditRepo *postgres.AuditRepository
}

func NewAdminHandler(userRepo *postgres.UserRepository, auditRepo *postgres.AuditRepository) *AdminHandler {
	return &AdminHandler{userRepo: userRepo, auditRepo: auditRepo}
}

// AdminUserResponse — публичный контракт пользователя для admin UI.
// ShlinkAPIKey НИКОГДА не включается.
type AdminUserResponse struct {
	ID         string `json:"id"`
	Sub        string `json:"sub"`
	Username   string `json:"username"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	SlugPrefix string `json:"slugPrefix"`
	Status     string `json:"status"`
	HasAPIKey  bool   `json:"hasApiKey"`
	CreatedAt  string `json:"createdAt"`
}

func toAdminUserResponse(u *domain.User) AdminUserResponse {
	return AdminUserResponse{
		ID:         u.ID.String(),
		Sub:        u.Sub,
		Username:   u.Username,
		Email:      u.Email,
		Role:       string(u.Role),
		SlugPrefix: u.SlugPrefix,
		Status:     string(u.Status),
		HasAPIKey:  u.ShlinkAPIKey != "",
		CreatedAt:  u.CreatedAt.Format(time.RFC3339),
	}
}

// GET /api/admin/users
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.userRepo.ListAll(r.Context())
	if err != nil {
		slog.Error("admin: list users failed", "err", err)
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}

	resp := make([]AdminUserResponse, 0, len(users))
	for _, u := range users {
		resp = append(resp, toAdminUserResponse(u))
	}
	writeJSON(w, resp, http.StatusOK)
}

// GET /api/admin/users/{sub}
func (h *AdminHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "sub")
	user, err := h.userRepo.GetBySub(r.Context(), sub)
	if err != nil || user == nil {
		writeJSON(w, map[string]string{"error": "not found"}, http.StatusNotFound)
		return
	}
	writeJSON(w, toAdminUserResponse(user), http.StatusOK)
}

// PUT /api/admin/users/{sub}
func (h *AdminHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "sub")

	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSON(w, map[string]string{"error": "bad request"}, http.StatusBadRequest)
		return
	}

	var payload struct {
		Role       *string `json:"role"`
		Status     *string `json:"status"`
		SlugPrefix *string `json:"slugPrefix"`
	}
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		writeJSON(w, map[string]string{"error": "invalid json"}, http.StatusBadRequest)
		return
	}

	fields := map[string]any{}
	if payload.Role != nil {
		fields["role"] = *payload.Role
	}
	if payload.Status != nil {
		fields["status"] = *payload.Status
	}
	if payload.SlugPrefix != nil {
		fields["slug_prefix"] = *payload.SlugPrefix
	}

	if err := h.userRepo.UpdateBySubFields(r.Context(), sub, fields); err != nil {
		slog.Error("admin: update user failed", "sub", sub, "err", err)
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}

	caller := middleware.UserFromCtx(r.Context())
	go h.auditRepo.Record(r.Context(), &domain.AuditEntry{
		UserSub:  caller.Sub,
		Username: caller.Username,
		Role:     string(caller.Role),
		Action:   "admin_update_user",
		Resource: r.URL.Path,
		Result:   "success",
		Details:  map[string]any{"target_sub": sub, "fields": fields},
	})

	writeJSON(w, map[string]string{"status": "updated"}, http.StatusOK)
}

// PUT /api/admin/users/{sub}/apikey
func (h *AdminHandler) UpdateAPIKey(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "sub")

	var payload struct {
		APIKey string `json:"apiKey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.APIKey == "" {
		writeJSON(w, map[string]string{"error": "apiKey required"}, http.StatusBadRequest)
		return
	}

	if err := h.userRepo.UpdateAPIKey(r.Context(), sub, payload.APIKey); err != nil {
		slog.Error("admin: update api key failed", "sub", sub, "err", err)
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}

	caller := middleware.UserFromCtx(r.Context())
	go h.auditRepo.Record(r.Context(), &domain.AuditEntry{
		UserSub:  caller.Sub,
		Username: caller.Username,
		Role:     string(caller.Role),
		Action:   "admin_update_apikey",
		Resource: r.URL.Path,
		Result:   "success",
		// Новый ключ в аудит НЕ пишем — sanitizeDetails уберёт его в любом случае
		Details: map[string]any{"target_sub": sub},
	})

	writeJSON(w, map[string]string{"status": "updated"}, http.StatusOK)
}

// PUT /api/admin/users/{sub}/prefix
func (h *AdminHandler) UpdateSlugPrefix(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "sub")

	var payload struct {
		Prefix string `json:"prefix"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSON(w, map[string]string{"error": "invalid json"}, http.StatusBadRequest)
		return
	}

	if err := h.userRepo.UpdateSlugPrefix(r.Context(), sub, payload.Prefix); err != nil {
		slog.Error("admin: update slug prefix failed", "sub", sub, "err", err)
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}

	caller := middleware.UserFromCtx(r.Context())
	go h.auditRepo.Record(r.Context(), &domain.AuditEntry{
		UserSub:  caller.Sub,
		Username: caller.Username,
		Role:     string(caller.Role),
		Action:   "admin_update_prefix",
		Resource: r.URL.Path,
		Result:   "success",
		Details:  map[string]any{"target_sub": sub, "prefix": payload.Prefix},
	})

	writeJSON(w, map[string]string{"status": "updated"}, http.StatusOK)
}

// GET /api/admin/users/{sub}/links
func (h *AdminHandler) GetUserLinks(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "sub")
	user, err := h.userRepo.GetBySub(r.Context(), sub)
	if err != nil || user == nil {
		writeJSON(w, map[string]string{"error": "user not found"}, http.StatusNotFound)
		return
	}

	// Запрашиваем ссылки от имени пользователя (с его ключом)
	// Это даёт правильный scope данных
	writeJSON(w, map[string]string{"message": "use /api/shlink/short-urls with user context"}, http.StatusOK)
}

// GET /api/admin/logs
func (h *AdminHandler) ListLogs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	page, _ := strconv.Atoi(q.Get("page"))
	if page <= 0 {
		page = 1
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = 50
	}

	f := postgres.AuditFilter{
		Username: q.Get("username"),
		Action:   q.Get("action"),
		Result:   q.Get("result"),
		Page:     page,
		Limit:    limit,
	}

	if df := q.Get("dateFrom"); df != "" {
		t, err := time.Parse("2006-01-02", df)
		if err == nil {
			f.DateFrom = &t
		}
	}
	if dt := q.Get("dateTo"); dt != "" {
		t, err := time.Parse("2006-01-02", dt)
		if err == nil {
			end := t.Add(24 * time.Hour)
			f.DateTo = &end
		}
	}

	page2, err := h.auditRepo.List(r.Context(), f)
	if err != nil {
		slog.Error("admin: list logs failed", "err", err)
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]any{
		"logs":  page2.Logs,
		"total": page2.Total,
	}, http.StatusOK)
}
