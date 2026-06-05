package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"unified-backend/internal/config"
	"unified-backend/internal/domain"
	"unified-backend/internal/middleware"
	"unified-backend/internal/repository/postgres"
	"unified-backend/internal/service"
)

type ShlinkProxyHandler struct {
	shlinkSvc *service.ShlinkService
	auditRepo *postgres.AuditRepository
	ownerRepo *postgres.URLOwnershipRepository
	cfg       *config.Config
}

func NewShlinkProxyHandler(
	svc *service.ShlinkService,
	auditRepo *postgres.AuditRepository,
	ownerRepo *postgres.URLOwnershipRepository,
	cfg *config.Config,
) *ShlinkProxyHandler {
	return &ShlinkProxyHandler{
		shlinkSvc: svc,
		auditRepo: auditRepo,
		ownerRepo: ownerRepo,
		cfg:       cfg,
	}
}

// GET /api/shlink/short-urls
func (h *ShlinkProxyHandler) ListShortURLs(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}
	p := h.shlinkSvc.Perms(user)
	if !p.CanViewOwnLinks && !p.CanViewAllLinks {
		h.recordAudit(r, user, "list_short_urls", "denied", map[string]any{"reason": "no view permission"})
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}

	resp, err := h.shlinkSvc.Client().GetShortURLs(r.Context(), user.ShlinkAPIKey, r.URL.RawQuery)
	if err != nil {
		slog.Error("proxy: get short-urls failed", "sub", user.Sub, "err", err)
		h.recordAudit(r, user, "list_short_urls", "error", map[string]any{"err": err.Error()})
		writeJSON(w, map[string]string{"error": "shlink unavailable"}, http.StatusBadGateway)
		return
	}

	var ownedCodes map[string]struct{}
	if !p.CanViewAllLinks {
		ownedCodes, _ = h.ownerRepo.GetShortCodeSet(r.Context(), user.Sub)
	}

	resp.ShortURLs.Data = h.shlinkSvc.FilterShortURLsByUser(resp.ShortURLs.Data, user, ownedCodes)

	h.recordAudit(r, user, "list_short_urls", "success", nil)
	writeJSON(w, resp, http.StatusOK)
}

// POST /api/shlink/short-urls
func (h *ShlinkProxyHandler) CreateShortURL(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSON(w, map[string]string{"error": "bad request"}, http.StatusBadRequest)
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
		writeJSON(w, map[string]string{"error": err.Error()}, http.StatusBadGateway)
		return
	}

	if err := h.ownerRepo.Save(r.Context(), result.ShortCode, user.Sub, ""); err != nil {
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
		writeJSON(w, map[string]string{"error": err.Error()}, http.StatusBadGateway)
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

	if err := h.checkModifyPermission(r.Context(), user, shortCode, true); err != nil {
		slog.Warn("proxy: delete denied", "sub", user.Sub, "shortCode", shortCode, "err", err)
		h.recordAudit(r, user, "delete_short_url", "denied", map[string]any{"shortCode": shortCode, "reason": err.Error()})
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}

	tombstoneURL := h.cfg.ShlinkDefaultDomain + "/gone"
	tombstoneBody, _ := json.Marshal(map[string]any{
		"longUrl":   tombstoneURL,
		"crawlable": false,
	})
	if _, err := h.shlinkSvc.Client().UpdateShortURL(
		r.Context(), user.ShlinkAPIKey, shortCode, bytes.NewReader(tombstoneBody),
	); err != nil {
		slog.Error("proxy: tombstone patch failed", "sub", user.Sub, "shortCode", shortCode, "err", err)
	}

	if err := h.ownerRepo.SoftDelete(r.Context(), shortCode, "", user.Sub); err != nil {
		slog.Error("proxy: soft delete failed", "sub", user.Sub, "shortCode", shortCode, "err", err)
		h.recordAudit(r, user, "delete_short_url", "error", map[string]any{"shortCode": shortCode, "err": err.Error()})
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}

	h.recordAudit(r, user, "delete_short_url", "success", map[string]any{"shortCode": shortCode})
	w.WriteHeader(http.StatusNoContent)
}

func (h *ShlinkProxyHandler) checkModifyPermission(
	ctx context.Context,
	user *domain.User,
	shortCode string,
	isDelete bool,
) error {
	canAll, canOwn := h.shlinkSvc.CanModifyShortCodeByPerms(user, isDelete)
	if canAll {
		return nil
	}
	if !canOwn {
		return errors.New("no edit/delete permission for role")
	}

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

	resp, err := h.shlinkSvc.Client().GetTags(r.Context(), user.ShlinkAPIKey)
	if err != nil {
		slog.Error("proxy: list tags failed", "sub", user.Sub, "err", err)
		writeJSON(w, map[string]string{"error": "shlink unavailable"}, http.StatusBadGateway)
		return
	}

	h.recordAudit(r, user, "list_tags", "success", nil)
	writeJSON(w, resp, http.StatusOK)
}

// PUT /api/shlink/tags
// Body: {"oldName": "...", "newName": "..."}
// Требует CanManageAllTags (глобальное управление тегами).
func (h *ShlinkProxyHandler) RenameTag(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}

	p := h.shlinkSvc.Perms(user)
	if !p.CanManageAllTags {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSON(w, map[string]string{"error": "bad request"}, http.StatusBadRequest)
		return
	}

	// shlink PUT /rest/v3/tags: body {"oldName":"...","newName":"..."}, returns error only
	if err := h.shlinkSvc.Client().RenameTag(r.Context(), user.ShlinkAPIKey, bytes.NewReader(bodyBytes)); err != nil {
		slog.Error("proxy: rename tag failed", "sub", user.Sub, "err", err)
		writeJSON(w, map[string]string{"error": err.Error()}, http.StatusBadGateway)
		return
	}

	// Extract oldName for audit log
	var names struct{ OldName string `json:"oldName"` }
	_ = json.Unmarshal(bodyBytes, &names)
	h.recordAudit(r, user, "rename_tag", "success", map[string]any{"oldName": names.OldName})
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /api/shlink/tags/{tagName}
// Требует CanManageAllTags.
func (h *ShlinkProxyHandler) DeleteTag(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}

	p := h.shlinkSvc.Perms(user)
	if !p.CanManageAllTags {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}

	tagName := chi.URLParam(r, "tagName")
	if tagName == "" {
		writeJSON(w, map[string]string{"error": "tagName required"}, http.StatusBadRequest)
		return
	}

	// shlink DELETE /rest/v3/tags?tags[]=... accepts a slice
	if err := h.shlinkSvc.Client().DeleteTags(r.Context(), user.ShlinkAPIKey, []string{tagName}); err != nil {
		slog.Error("proxy: delete tag failed", "sub", user.Sub, "tag", tagName, "err", err)
		writeJSON(w, map[string]string{"error": err.Error()}, http.StatusBadGateway)
		return
	}

	h.recordAudit(r, user, "delete_tag", "success", map[string]any{"tag": tagName})
	w.WriteHeader(http.StatusNoContent)
}

// recordAudit — тонкая обёртка над AuditRepository.Record.
// Не блокирует обработчик: ошибка логируется внутри Record.
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
	details := make(map[string]any, len(extra)+3)
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
