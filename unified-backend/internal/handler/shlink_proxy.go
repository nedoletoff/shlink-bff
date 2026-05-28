package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"unified-backend/internal/domain"
	"unified-backend/internal/middleware"
	"unified-backend/internal/repository/postgres"
	"unified-backend/internal/service"
)

type ShlinkProxyHandler struct {
	shlinkSvc *service.ShlinkService
	auditRepo *postgres.AuditRepository
}

func NewShlinkProxyHandler(
	svc *service.ShlinkService,
	auditRepo *postgres.AuditRepository,
) *ShlinkProxyHandler {
	return &ShlinkProxyHandler{shlinkSvc: svc, auditRepo: auditRepo}
}

// GET /api/shlink/short-urls
func (h *ShlinkProxyHandler) ListShortURLs(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
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

	// Для role=user — дополнительно фильтруем по префиксу (если включён)
	urls := resp.ShortURLs.Data
	if user.Role == domain.RoleUser {
		urls = h.shlinkSvc.FilterShortURLsByUser(urls, user)
	}
	resp.ShortURLs.Data = urls

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

	// Enforce slug prefix для role=user
	var customSlug *string
	if cs, ok := payload["customSlug"].(string); ok && cs != "" {
		customSlug = &cs
	}

	enforced, err := h.shlinkSvc.EnforceSlugPrefix(r.Context(), user, customSlug)
	if err != nil {
		slog.Warn("proxy: slug prefix violation", "sub", user.Sub, "err", err)
		h.recordAudit(r, user, "create_short_url", "denied", map[string]any{"reason": err.Error()})
		writeJSON(w, map[string]string{"error": err.Error()}, http.StatusBadRequest)
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

	if err := h.shlinkSvc.Client().DeleteShortURL(r.Context(), user.ShlinkAPIKey, shortCode); err != nil {
		slog.Error("proxy: delete short-url failed", "sub", user.Sub, "shortCode", shortCode, "err", err)
		h.recordAudit(r, user, "delete_short_url", "error", map[string]any{"shortCode": shortCode, "err": err.Error()})
		writeJSON(w, map[string]string{"error": err.Error()}, http.StatusBadGateway)
		return
	}

	h.recordAudit(r, user, "delete_short_url", "success", map[string]any{"shortCode": shortCode})
	w.WriteHeader(http.StatusNoContent)
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
		slog.Error("proxy: get tags failed", "sub", user.Sub, "err", err)
		writeJSON(w, map[string]string{"error": "shlink unavailable"}, http.StatusBadGateway)
		return
	}

	writeJSON(w, resp, http.StatusOK)
}

// POST /api/shlink/tags — создаёт тег через Shlink (Shlink создаёт теги при добавлении к ссылке)
// Здесь используем rename как "создание" нового тега
func (h *ShlinkProxyHandler) CreateTag(w http.ResponseWriter, r *http.Request) {
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

	if err := h.shlinkSvc.Client().RenameTag(r.Context(), user.ShlinkAPIKey, bytes.NewReader(bodyBytes)); err != nil {
		slog.Error("proxy: create tag failed", "sub", user.Sub, "err", err)
		writeJSON(w, map[string]string{"error": err.Error()}, http.StatusBadGateway)
		return
	}

	h.recordAudit(r, user, "create_tag", "success", nil)
	w.WriteHeader(http.StatusCreated)
}

// PUT /api/shlink/tags/{tagId} — переименование тега
func (h *ShlinkProxyHandler) RenameTag(w http.ResponseWriter, r *http.Request) {
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

	if err := h.shlinkSvc.Client().RenameTag(r.Context(), user.ShlinkAPIKey, bytes.NewReader(bodyBytes)); err != nil {
		slog.Error("proxy: rename tag failed", "sub", user.Sub, "err", err)
		writeJSON(w, map[string]string{"error": err.Error()}, http.StatusBadGateway)
		return
	}

	h.recordAudit(r, user, "rename_tag", "success", nil)
	writeJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
}

// DELETE /api/shlink/tags/{tagId}
func (h *ShlinkProxyHandler) DeleteTag(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}

	tagName := chi.URLParam(r, "tagId")
	if tagName == "" {
		writeJSON(w, map[string]string{"error": "tagId required"}, http.StatusBadRequest)
		return
	}

	if err := h.shlinkSvc.Client().DeleteTags(r.Context(), user.ShlinkAPIKey, []string{tagName}); err != nil {
		slog.Error("proxy: delete tag failed", "sub", user.Sub, "tag", tagName, "err", err)
		writeJSON(w, map[string]string{"error": err.Error()}, http.StatusBadGateway)
		return
	}

	h.recordAudit(r, user, "delete_tag", "success", map[string]any{"tag": tagName})
	w.WriteHeader(http.StatusNoContent)
}

// recordAudit — вспомогательная запись в аудит
func (h *ShlinkProxyHandler) recordAudit(
	r *http.Request,
	user *domain.User,
	action, result string,
	details map[string]any,
) {
	go h.auditRepo.Record(r.Context(), &domain.AuditEntry{
		UserSub:   user.Sub,
		Username:  user.Username,
		Role:      string(user.Role),
		Action:    action,
		Resource:  r.URL.Path,
		Result:    result,
		Details:   details,
		IPAddress: r.RemoteAddr,
		UserAgent: r.Header.Get("User-Agent"),
	})
}
