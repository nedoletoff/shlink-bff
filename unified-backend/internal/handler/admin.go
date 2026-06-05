package handler

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"unified-backend/internal/domain"
	"unified-backend/internal/middleware"
	"unified-backend/internal/repository/postgres"
	"unified-backend/internal/service"
)

type AdminHandler struct {
	userRepo  *postgres.UserRepository
	auditRepo *postgres.AuditRepository
	shlinkSvc *service.ShlinkService
}

func NewAdminHandler(userRepo *postgres.UserRepository, auditRepo *postgres.AuditRepository, shlinkSvc *service.ShlinkService) *AdminHandler {
	return &AdminHandler{userRepo: userRepo, auditRepo: auditRepo, shlinkSvc: shlinkSvc}
}

// recordAuditAsync — записи аудита в горутине с детачнутым контекстом (#4):
// r.Context() отменяется после возврата handler, и записи теряются.
func (h *AdminHandler) recordAuditAsync(entry *domain.AuditEntry) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		h.auditRepo.Record(ctx, entry)
	}()
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

// UserDetailResponse — расширенный ответ для страницы деталей пользователя.
// Включает базовую инфу + агрегированные данные по ссылкам из shlink.
type UserDetailResponse struct {
	Sub            string            `json:"sub"`
	Username       string            `json:"username"`
	Email          string            `json:"email"`
	Role           string            `json:"role"`
	SlugPrefix     string            `json:"slugPrefix"`
	Status         string            `json:"status"`
	HasAPIKey      bool              `json:"hasApiKey"`
	CreatedAt      string            `json:"createdAt"`
	LinksCount     int               `json:"linksCount"`
	VisitsTotal    int               `json:"visitsTotal"`
	ActivityPerDay []adminClickPoint `json:"activityPerDay"`
	Links          []adminShortURL   `json:"links"`
}

type adminClickPoint struct {
	Date   string `json:"date"`
	Clicks int    `json:"clicks"`
}

type adminShortURL struct {
	ShortCode   string   `json:"shortCode"`
	ShortURL    string   `json:"shortUrl"`
	LongURL     string   `json:"longUrl"`
	Title       string   `json:"title"`
	Tags        []string `json:"tags"`
	DateCreated string   `json:"dateCreated"`
	VisitsTotal int      `json:"visitsTotal"`
}

// GET /api/admin/users/{sub}
func (h *AdminHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "sub")
	user, err := h.userRepo.GetBySub(r.Context(), sub)
	if err != nil || user == nil {
		writeJSON(w, map[string]string{"error": "not found"}, http.StatusNotFound)
		return
	}

	// Если у пользователя нет API-ключа — возвращаем базовую инфу без shlink-данных
	if user.ShlinkAPIKey == "" {
		writeJSON(w, UserDetailResponse{
			Sub:            user.Sub,
			Username:       user.Username,
			Email:          user.Email,
			Role:           string(user.Role),
			SlugPrefix:     user.SlugPrefix,
			Status:         string(user.Status),
			HasAPIKey:      false,
			CreatedAt:      user.CreatedAt.Format(time.RFC3339),
			LinksCount:     0,
			VisitsTotal:    0,
			ActivityPerDay: []adminClickPoint{},
			Links:          []adminShortURL{},
		}, http.StatusOK)
		return
	}

	// Запрашиваем ссылки от имени пользователя (с его ключом)
	urlsResp, err := h.shlinkSvc.Client().GetShortURLs(r.Context(), user.ShlinkAPIKey, "itemsPerPage=1000")
	if err != nil {
		slog.Warn("admin: get user links failed", "sub", sub, "err", err)
	}

	const dayFmt = "2006-01-02"
	now := time.Now()
	period := 30
	buckets := make(map[string]int, period)
	ordered := make([]string, 0, period)
	for i := period - 1; i >= 0; i-- {
		d := now.AddDate(0, 0, -i).Format(dayFmt)
		buckets[d] = 0
		ordered = append(ordered, d)
	}

	links := []adminShortURL{}
	visitsTotal := 0

	if urlsResp != nil {
		for _, u := range urlsResp.ShortURLs.Data {
			tags := u.Tags
			if tags == nil {
				tags = []string{}
			}
			links = append(links, adminShortURL{
				ShortCode:   u.ShortCode,
				ShortURL:    u.ShortURL,
				LongURL:     u.LongURL,
				Title:       u.Title,
				Tags:        tags,
				DateCreated: u.DateCreated,
				VisitsTotal: u.VisitsSummary.Total,
			})
			visitsTotal += u.VisitsSummary.Total
			if t, e := time.Parse(time.RFC3339, u.DateCreated); e == nil {
				d := t.Format(dayFmt)
				if _, ok := buckets[d]; ok {
					buckets[d]++
				}
			}
		}
		sort.Slice(links, func(i, j int) bool {
			return links[i].VisitsTotal > links[j].VisitsTotal
		})
	}

	activity := make([]adminClickPoint, 0, period)
	for _, d := range ordered {
		activity = append(activity, adminClickPoint{Date: d, Clicks: buckets[d]})
	}

	writeJSON(w, UserDetailResponse{
		Sub:            user.Sub,
		Username:       user.Username,
		Email:          user.Email,
		Role:           string(user.Role),
		SlugPrefix:     user.SlugPrefix,
		Status:         string(user.Status),
		HasAPIKey:      true,
		CreatedAt:      user.CreatedAt.Format(time.RFC3339),
		LinksCount:     len(links),
		VisitsTotal:    visitsTotal,
		ActivityPerDay: activity,
		Links:          links,
	}, http.StatusOK)
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

	actor := middleware.ClaimsFromContext(r.Context())
	h.recordAuditAsync(&domain.AuditEntry{
		ActorSub:   actor.Sub,
		TargetSub:  sub,
		Action:     "admin.user.update",
		ReqBody:    string(bodyBytes),
		StatusCode: http.StatusOK,
	})

	updated, _ := h.userRepo.GetBySub(r.Context(), sub)
	if updated == nil {
		writeJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
		return
	}
	writeJSON(w, toAdminUserResponse(updated), http.StatusOK)
}

// PUT /api/admin/users/{sub}/apikey
func (h *AdminHandler) UpdateAPIKey(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "sub")

	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSON(w, map[string]string{"error": "bad request"}, http.StatusBadRequest)
		return
	}

	var payload struct {
		APIKey string `json:"apiKey"`
	}
	if err := json.Unmarshal(bodyBytes, &payload); err != nil || payload.APIKey == "" {
		writeJSON(w, map[string]string{"error": "apiKey required"}, http.StatusBadRequest)
		return
	}

	if err := h.userRepo.UpdateBySubFields(r.Context(), sub, map[string]any{
		"shlink_api_key": payload.APIKey,
	}); err != nil {
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}

	actor := middleware.ClaimsFromContext(r.Context())
	h.recordAuditAsync(&domain.AuditEntry{
		ActorSub:   actor.Sub,
		TargetSub:  sub,
		Action:     "admin.user.apikey.update",
		ReqBody:    `{"apiKey":"[redacted]"}`,
		StatusCode: http.StatusOK,
	})

	writeJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
}

// PUT /api/admin/users/{sub}/prefix
func (h *AdminHandler) UpdateSlugPrefix(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "sub")

	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSON(w, map[string]string{"error": "bad request"}, http.StatusBadRequest)
		return
	}

	var payload struct {
		Prefix string `json:"prefix"`
	}
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		writeJSON(w, map[string]string{"error": "invalid json"}, http.StatusBadRequest)
		return
	}

	if err := h.userRepo.UpdateBySubFields(r.Context(), sub, map[string]any{
		"slug_prefix": payload.Prefix,
	}); err != nil {
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}

	actor := middleware.ClaimsFromContext(r.Context())
	h.recordAuditAsync(&domain.AuditEntry{
		ActorSub:   actor.Sub,
		TargetSub:  sub,
		Action:     "admin.user.prefix.update",
		ReqBody:    string(bodyBytes),
		StatusCode: http.StatusOK,
	})

	writeJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
}

// GET /api/admin/users/{sub}/links — устаревший endpoint, оставлен для совместимости
func (h *AdminHandler) GetUserLinks(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{
		"message": "use GET /api/admin/users/{sub} — links are included in the response",
	}, http.StatusOK)
}

// GET /api/admin/logs
func (h *AdminHandler) ListLogs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit := 50
	if l := q.Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 500 {
			limit = v
		}
	}
	offset := 0
	if o := q.Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	entries, err := h.auditRepo.List(r.Context(), limit, offset)
	if err != nil {
		slog.Error("admin: list logs failed", "err", err)
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}
	writeJSON(w, entries, http.StatusOK)
}

// GET /api/admin/roles
func (h *AdminHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.userRepo.ListRolePermissions(r.Context())
	if err != nil {
		slog.Error("admin: list roles failed", "err", err)
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}
	writeJSON(w, roles, http.StatusOK)
}

// GET /api/admin/roles/{role}
func (h *AdminHandler) GetRole(w http.ResponseWriter, r *http.Request) {
	role := chi.URLParam(r, "role")
	rp, err := h.userRepo.GetRolePermissions(r.Context(), role)
	if err != nil || rp == nil {
		writeJSON(w, map[string]string{"error": "not found"}, http.StatusNotFound)
		return
	}
	writeJSON(w, rp, http.StatusOK)
}

// PUT /api/admin/roles/{role}/permissions
func (h *AdminHandler) UpdateRolePermissions(w http.ResponseWriter, r *http.Request) {
	role := chi.URLParam(r, "role")

	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSON(w, map[string]string{"error": "bad request"}, http.StatusBadRequest)
		return
	}

	var perms domain.RolePermissions
	if err := json.Unmarshal(bodyBytes, &perms); err != nil {
		writeJSON(w, map[string]string{"error": "invalid json"}, http.StatusBadRequest)
		return
	}
	perms.Role = role

	if err := h.userRepo.UpsertRolePermissions(r.Context(), &perms); err != nil {
		slog.Error("admin: update role perms failed", "role", role, "err", err)
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}

	actor := middleware.ClaimsFromContext(r.Context())
	h.recordAuditAsync(&domain.AuditEntry{
		ActorSub:   actor.Sub,
		Action:     "admin.role.permissions.update",
		ReqBody:    string(bodyBytes),
		StatusCode: http.StatusOK,
	})

	writeJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
}
