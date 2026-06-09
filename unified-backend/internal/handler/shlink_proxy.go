package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"unified-backend/internal/config"
	"unified-backend/internal/controller"
	"unified-backend/internal/domain"
	"unified-backend/internal/middleware"
	"unified-backend/internal/service"
)

type ShlinkProxyHandler struct {
	shlinkSvc *service.ShlinkService
	auditRepo AuditRepo
	ownerRepo OwnershipRepo
	cfg       *config.Config
	permCtrl  controller.PermChecker
}

func NewShlinkProxyHandler(
	svc *service.ShlinkService,
	auditRepo AuditRepo,
	ownerRepo OwnershipRepo,
	cfg *config.Config,
	permCtrl controller.PermChecker,
) *ShlinkProxyHandler {
	return &ShlinkProxyHandler{
		shlinkSvc: svc,
		auditRepo: auditRepo,
		ownerRepo: ownerRepo,
		cfg:       cfg,
		permCtrl:  permCtrl,
	}
}

// createShortURLRequest используется только для BFF-валидации;
// само тело проксируется без изменений (кроме customSlug enforcement).
type createShortURLRequest struct {
	LongURL    string   `json:"longUrl"`
	Title      string   `json:"title"`
	CustomSlug string   `json:"customSlug"`
	Tags       []string `json:"tags"`
	MaxVisits  *int     `json:"maxVisits"`
	ValidSince *string  `json:"validSince"`
	ValidUntil *string  `json:"validUntil"`
}

func validateCreateShortURLPayload(req *createShortURLRequest) error {
	if req.MaxVisits != nil && *req.MaxVisits < 1 {
		return fmt.Errorf("maxVisits must be >= 1")
	}
	if req.ValidSince != nil && req.ValidUntil != nil {
		since, err1 := time.Parse(time.RFC3339, *req.ValidSince)
		until, err2 := time.Parse(time.RFC3339, *req.ValidUntil)
		if err1 != nil {
			return fmt.Errorf("validSince: invalid RFC3339 format")
		}
		if err2 != nil {
			return fmt.Errorf("validUntil: invalid RFC3339 format")
		}
		if !since.Before(until) {
			return fmt.Errorf("validSince must be before validUntil")
		}
	}
	if len(req.Tags) > 20 {
		return fmt.Errorf("tags: max 20 tags allowed")
	}
	for _, tag := range req.Tags {
		if len(tag) > 64 {
			return fmt.Errorf("tags: each tag must be <= 64 characters")
		}
	}
	return nil
}

// GET /api/shlink/short-urls
func (h *ShlinkProxyHandler) ListShortURLs(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}

	canViewAll, err := h.permCtrl.Check(r.Context(), user.ID, domain.PermShortURLsViewAll)
	if err != nil {
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}
	canCreate, err := h.permCtrl.Check(r.Context(), user.ID, domain.PermShortURLsCreate)
	if err != nil {
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}
	if !canViewAll && !canCreate {
		h.recordAudit(r, user, "list_short_urls", "denied", map[string]any{"reason": "no view permission"})
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}

	statusFilter := r.URL.Query().Get("status")
	if statusFilter == "" {
		statusFilter = "active"
	}

	resp, err := h.shlinkSvc.Client().GetShortURLs(r.Context(), user.ShlinkAPIKey, r.URL.RawQuery)
	if err != nil {
		slog.Error("proxy: get short-urls failed", "sub", user.Sub, "err", err)
		h.recordAudit(r, user, "list_short_urls", "error", map[string]any{"err": err.Error()})
		writeJSON(w, map[string]string{"error": "shlink unavailable"}, http.StatusBadGateway)
		return
	}

	if !canViewAll {
		ownedCodes, _ := h.ownerRepo.GetShortCodeSet(r.Context(), user.Sub)
		resp.ShortURLs.Data = h.shlinkSvc.FilterShortURLsByUser(resp.ShortURLs.Data, user, ownedCodes)
	}

	if statusFilter != "all" {
		statusMap, _ := h.ownerRepo.GetStatusCodeSet(r.Context(), user.Sub)
		wantActive := statusFilter == "active"
		filtered := resp.ShortURLs.Data[:0]
		for _, u := range resp.ShortURLs.Data {
			active, known := statusMap[u.ShortCode]
			if !known {
				active = true
			}
			if active == wantActive {
				filtered = append(filtered, u)
			}
		}
		resp.ShortURLs.Data = filtered
	}

	n := len(resp.ShortURLs.Data)
	if !canViewAll {
		resp.ShortURLs.Pagination.TotalItems = n
		resp.ShortURLs.Pagination.ItemsInCurrentPage = n
		resp.ShortURLs.Pagination.PagesCount = 1
		resp.ShortURLs.Pagination.CurrentPage = 1
	}

	h.recordAudit(r, user, "list_short_urls", "success", map[string]any{"status": statusFilter})
	writeJSON(w, resp, http.StatusOK)
}

// POST /api/shlink/short-urls
func (h *ShlinkProxyHandler) CreateShortURL(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}

	ok, err := h.permCtrl.Check(r.Context(), user.ID, domain.PermShortURLsCreate)
	if err != nil {
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}
	if !ok {
		h.recordAudit(r, user, "create_short_url", "denied", map[string]any{"reason": "no short_urls.create permission"})
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSON(w, map[string]string{"error": "bad request"}, http.StatusBadRequest)
		return
	}

	var req createShortURLRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		writeJSON(w, map[string]string{"error": "invalid json"}, http.StatusBadRequest)
		return
	}
	if err := validateCreateShortURLPayload(&req); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, http.StatusUnprocessableEntity)
		return
	}

	var payload map[string]any
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		writeJSON(w, map[string]string{"error": "invalid json"}, http.StatusBadRequest)
		return
	}

	var customSlug *string
	if cs, ok := payload["customSlug"].(string); ok && cs != "" {
		customSlug = &cs
	}

	enforced, err := h.shlinkSvc.EnforceSlugPrefix(r.Context(), user, customSlug)
	if err != nil {
		slog.Warn("proxy: slug enforcement failed", "sub", user.Sub, "err", err)
		h.recordAudit(r, user, "create_short_url", "denied", map[string]any{"reason": err.Error()})
		writeJSON(w, map[string]string{"error": err.Error()}, http.StatusForbidden)
		return
	}
	if enforced != "" {
		payload["customSlug"] = enforced
	}

	modifiedBody, _ := json.Marshal(payload)

	result, err := h.shlinkSvc.Client().CreateShortURL(
		r.Context(), user.ShlinkAPIKey, bytes.NewReader(modifiedBody),
	)
	if err != nil {
		slog.Error("proxy: create short-url failed", "sub", user.Sub, "err", err)
		h.recordAudit(r, user, "create_short_url", "error", map[string]any{"err": err.Error()})
		writeJSON(w, map[string]string{"error": friendlyShlinkError(err)}, http.StatusBadRequest)
		return
	}

	if err := h.ownerRepo.Save(r.Context(), result.ShortCode, user.Sub, user.Username, ""); err != nil {
		slog.Error("proxy: failed to save url ownership", "sub", user.Sub, "shortCode", result.ShortCode, "err", err)
	}

	h.recordAudit(r, user, "create_short_url", "success", map[string]any{"shortCode": result.ShortCode})
	writeJSON(w, result, http.StatusCreated)
}

// PATCH /api/shlink/short-urls/{shortCode}
func (h *ShlinkProxyHandler) UpdateShortURL(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}

	shortCode := chi.URLParam(r, "shortCode")
	if shortCode == "" {
		writeJSON(w, map[string]string{"error": "shortCode required"}, http.StatusBadRequest)
		return
	}

	if err := h.checkModifyPermission(r.Context(), user, shortCode, false); err != nil {
		slog.Warn("proxy: update denied", "sub", user.Sub, "shortCode", shortCode, "err", err)
		h.recordAudit(r, user, "update_short_url", "denied", map[string]any{"shortCode": shortCode, "reason": err.Error()})
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSON(w, map[string]string{"error": "bad request"}, http.StatusBadRequest)
		return
	}

	result, err := h.shlinkSvc.Client().UpdateShortURL(
		r.Context(), user.ShlinkAPIKey, shortCode, bytes.NewReader(bodyBytes),
	)
	if err != nil {
		slog.Error("proxy: update short-url failed", "sub", user.Sub, "shortCode", shortCode, "err", err)
		h.recordAudit(r, user, "update_short_url", "error", map[string]any{"shortCode": shortCode, "err": err.Error()})
		writeJSON(w, map[string]string{"error": friendlyShlinkError(err)}, http.StatusBadRequest)
		return
	}

	h.recordAudit(r, user, "update_short_url", "success", map[string]any{"shortCode": shortCode})
	writeJSON(w, result, http.StatusOK)
}

// DELETE /api/shlink/short-urls/{shortCode}
func (h *ShlinkProxyHandler) DeleteShortURL(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}

	shortCode := chi.URLParam(r, "shortCode")
	if shortCode == "" {
		writeJSON(w, map[string]string{"error": "shortCode required"}, http.StatusBadRequest)
		return
	}

	canDeleteAll, err := h.permCtrl.Check(r.Context(), user.ID, domain.PermShortURLsDelete)
	if err != nil {
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}

	if !canDeleteAll {
		// soft-delete only — проверяем владение
		isOwner, ownerErr := h.ownerRepo.IsOwner(r.Context(), shortCode, "", user.Sub)
		if ownerErr != nil || !isOwner {
			h.recordAudit(r, user, "delete_short_url", "denied",
				map[string]any{"shortCode": shortCode, "reason": "not the owner or no delete permission"})
			writeJSON(w, map[string]string{"error": "forbidden: use /deactivate for soft delete"}, http.StatusForbidden)
			return
		}
	}

	if err := h.shlinkSvc.Client().DeleteShortURL(r.Context(), user.ShlinkAPIKey, shortCode); err != nil {
		slog.Error("proxy: shlink delete failed", "sub", user.Sub, "shortCode", shortCode, "err", err)
		h.recordAudit(r, user, "delete_short_url", "error", map[string]any{"shortCode": shortCode, "err": err.Error()})
		writeJSON(w, map[string]string{"error": friendlyShlinkError(err)}, http.StatusBadGateway)
		return
	}

	if err := h.ownerRepo.HardDelete(r.Context(), shortCode, ""); err != nil {
		slog.Error("proxy: hard delete ownership failed", "sub", user.Sub, "shortCode", shortCode, "err", err)
	}

	h.recordAudit(r, user, domain.ActionShortURLDeletedPermanently, "success", map[string]any{"shortCode": shortCode})
	w.WriteHeader(http.StatusNoContent)
}

func (h *ShlinkProxyHandler) checkModifyPermission(
	ctx context.Context,
	user *domain.User,
	shortCode string,
	isDelete bool,
) error {
	perm := domain.PermShortURLsUpdate
	if isDelete {
		perm = domain.PermShortURLsDelete
	}
	canAll, err := h.permCtrl.Check(ctx, user.ID, perm)
	if err != nil {
		return errors.New("permission check failed")
	}
	if canAll {
		return nil
	}
	// Фаллбэк: проверяем владение
	isOwner, err := h.ownerRepo.IsOwner(ctx, shortCode, "", user.Sub)
	if err != nil {
		slog.Error("proxy: ownership check failed", "sub", user.Sub, "shortCode", shortCode, "err", err)
		return errors.New("ownership check failed")
	}
	if !isOwner {
		return errors.New("not the owner")
	}
	return nil
}

// GET /api/shlink/tags
func (h *ShlinkProxyHandler) ListTags(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}

	// Листинг доступен всем авторизованным (dashboard.view суффициентно).
	resp, err := h.shlinkSvc.Client().GetTags(r.Context(), user.ShlinkAPIKey)
	if err != nil {
		slog.Error("proxy: list tags failed", "sub", user.Sub, "err", err)
		writeJSON(w, map[string]string{"error": "shlink unavailable"}, http.StatusBadGateway)
		return
	}

	h.recordAudit(r, user, "list_tags", "success", nil)
	writeJSON(w, resp, http.StatusOK)
}

// POST /api/shlink/tags — требует short_urls.create
func (h *ShlinkProxyHandler) CreateTag(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}

	ok, err := h.permCtrl.Check(r.Context(), user.ID, domain.PermShortURLsCreate)
	if err != nil {
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}
	if !ok {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
	if err != nil {
		writeJSON(w, map[string]string{"error": "bad request"}, http.StatusBadRequest)
		return
	}

	resp, err := h.shlinkSvc.Client().CreateTag(r.Context(), user.ShlinkAPIKey, bytes.NewReader(bodyBytes))
	if err != nil {
		slog.Error("proxy: create tag failed", "sub", user.Sub, "err", err)
		writeJSON(w, map[string]string{"error": err.Error()}, http.StatusBadGateway)
		return
	}

	h.recordAudit(r, user, "create_tag", "success", nil)
	writeJSON(w, resp, http.StatusCreated)
}

// PUT /api/shlink/tags/{tagId} — требует short_urls.view_all (rename = видит все теги)
func (h *ShlinkProxyHandler) RenameTag(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}

	ok, err := h.permCtrl.Check(r.Context(), user.ID, domain.PermShortURLsViewAll)
	if err != nil {
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}
	if !ok {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSON(w, map[string]string{"error": "bad request"}, http.StatusBadRequest)
		return
	}

	if err := h.shlinkSvc.Client().RenameTag(r.Context(), user.ShlinkAPIKey, bytes.NewReader(bodyBytes)); err != nil {
		slog.Error("proxy: rename tag failed", "sub", user.Sub, "err", err)
		writeJSON(w, map[string]string{"error": err.Error()}, http.StatusBadGateway)
		return
	}

	var names struct{ OldName string `json:"oldName"` }
	_ = json.Unmarshal(bodyBytes, &names)
	h.recordAudit(r, user, "rename_tag", "success", map[string]any{"oldName": names.OldName})
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/shlink/tags/{tagId} — фикс: chi.URLParam "tagId" (было "tagName"); требует short_urls.delete
func (h *ShlinkProxyHandler) DeleteTag(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}

	ok, err := h.permCtrl.Check(r.Context(), user.ID, domain.PermShortURLsDelete)
	if err != nil {
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}
	if !ok {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}

	// Фикс: было chi.URLParam(r, "tagName") — не работало
	tagID := chi.URLParam(r, "tagId")
	if tagID == "" {
		writeJSON(w, map[string]string{"error": "tagId required"}, http.StatusBadRequest)
		return
	}

	if err := h.shlinkSvc.Client().DeleteTags(r.Context(), user.ShlinkAPIKey, []string{tagID}); err != nil {
		slog.Error("proxy: delete tag failed", "sub", user.Sub, "tag", tagID, "err", err)
		writeJSON(w, map[string]string{"error": err.Error()}, http.StatusBadGateway)
		return
	}

	h.recordAudit(r, user, "delete_tag", "success", map[string]any{"tag": tagID})
	w.WriteHeader(http.StatusNoContent)
}

func friendlyShlinkError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	start := strings.Index(msg, "{")
	if start < 0 {
		return msg
	}
	var body struct {
		Type   string `json:"type"`
		Detail string `json:"detail"`
		Title  string `json:"title"`
	}
	if jsonErr := json.Unmarshal([]byte(msg[start:]), &body); jsonErr != nil {
		return msg
	}
	switch {
	case strings.Contains(body.Type, "non-unique-slug"):
		return "Ссылка с таким именем уже существует"
	case strings.Contains(body.Type, "invalid-url"):
		return "Указан некорректный URL"
	case strings.Contains(body.Type, "invalid-short-code-length"):
		return "Недопустимая длина кода ссылки"
	default:
		if body.Detail != "" {
			return body.Detail
		}
		return msg
	}
}

func (h *ShlinkProxyHandler) recordAudit(
	r *http.Request,
	user *domain.User,
	action, status string,
	extra map[string]any,
) {
	if h.auditRepo == nil {
		return
	}
	ip := r.RemoteAddr
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ip = xff
	}
	details := make(map[string]any, len(extra)+2)
	for k, v := range extra {
		details[k] = v
	}
	details["method"] = r.Method
	details["path"] = r.URL.Path

	h.auditRepo.Record(r.Context(), &domain.AuditEntry{
		UserSub:   user.Sub,
		Username:  user.Username,
		Role:      user.Role,
		Action:    action,
		Resource:  r.URL.Path,
		Result:    status,
		Details:   details,
		IPAddress: ip,
		UserAgent: r.Header.Get("User-Agent"),
		CreatedAt: time.Now(),
	})
}
