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
	"github.com/jackc/pgx/v5"

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

	// Для обычных пользователей (не CanViewAllLinks) фильтруем по ownership таблице.
	// Ссылки, созданные напрямую через shlink API (без записи в url_ownership),
	// не будут видны обычному пользователю — только в шlink-native панели или для admin.
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

	// Сохраняем ownership. Ошибка не критична — ссылка уже создана в shlink,
	// логируем и продолжаем. Запись появится без owner и будет видна только admin.
	domain_ := result.ShortURL // домен берём из shortUrl shlink-ответа
	if err := h.ownerRepo.Save(r.Context(), result.ShortCode, user.Sub, domain_); err != nil {
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
//
// Мы НЕ вызываем реальное удаление в shlink. Вместо этого:
//  1. Проверяем ownership.
//  2. PATCH в shlink: longUrl = tombstone, crawlable=false — ссылка "мертва",
//     но остаётся в любой shlink-панели (включая нативную), редиректит на /gone.
//  3. SoftDelete в url_ownership: записываем deleted_at + deleted_by.
//  4. 204 No Content.
//
// Благодаря этому:
//   - Ссылка видна в shlink-панели (в shlink она не удалена).
//   - В нашем BFF она отфильтровывается через GetShortCodeSet (deleted_at IS NOT NULL).
//   - История хранится: owner_sub, created_at, deleted_by, deleted_at.
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

	// PATCH tombstone в shlink.
	// Ссылка перестаёт работать как редирект, но остаётся в shlink (видна в shlink UI/API).
	tombstoneURL := h.cfg.ShlinkDefaultDomain + "/gone"
	tombstoneBody, _ := json.Marshal(map[string]any{
		"longUrl":   tombstoneURL,
		"crawlable": false,
	})
	if _, err := h.shlinkSvc.Client().UpdateShortURL(
		r.Context(), user.ShlinkAPIKey, shortCode, bytes.NewReader(tombstoneBody),
	); err != nil {
		// Tombstone не удался — не блокируем soft-delete в нашей БД,
		// но логируем как error чтобы было видно.
		slog.Error("proxy: tombstone patch failed", "sub", user.Sub, "shortCode", shortCode, "err", err)
	}

	// Soft-delete в url_ownership.
	if err := h.ownerRepo.SoftDelete(r.Context(), shortCode, "", user.Sub); err != nil {
		slog.Error("proxy: soft delete failed", "sub", user.Sub, "shortCode", shortCode, "err", err)
		h.recordAudit(r, user, "delete_short_url", "error", map[string]any{"shortCode": shortCode, "err": err.Error()})
		writeJSON(w, map[string]string{"error": "internal error"}, http.StatusInternalServerError)
		return
	}

	h.recordAudit(r, user, "delete_short_url", "success", map[string]any{"shortCode": shortCode})
	w.WriteHeader(http.StatusNoContent)
}

// checkModifyPermission проверяет права на изменение/удаление ссылки.
//
// Логика:
//  1. canAll (CanEditAllLinks / CanDeleteAllLinks) → разрешено сразу.
//  2. canOwn (CanEditOwnLinks / CanDeleteOwnLinks) → проверяем IsOwner в url_ownership.
//     Если записи в url_ownership нет (ссылка создана напрямую через shlink API)
//     → только canAll может изменить.
//  3. Иначе → 403.
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

	// Проверяем ownership. domain="" — используем пустой домен (дефолт).
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
	p := h.shlinkSvc.Perms(user)
	if !p.CanManageOwnTags && !p.CanManageAllTags {
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

// PUT /api/shlink/tags/{tagId}
func (h *ShlinkProxyHandler) RenameTag(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}
	p := h.shlinkSvc.Perms(user)
	if !p.CanManageOwnTags && !p.CanManageAllTags {
		h.recordAudit(r, user, "rename_tag", "denied", map[string]any{"reason": "no tag permission"})
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
	p := h.shlinkSvc.Perms(user)
	if !p.CanManageOwnTags && !p.CanManageAllTags {
		h.recordAudit(r, user, "delete_tag", "denied", map[string]any{"reason": "no tag permission"})
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

func (h *ShlinkProxyHandler) recordAudit(
	r *http.Request,
	user *domain.User,
	action, result string,
	details map[string]any,
) {
	entry := &domain.AuditEntry{
		UserSub:   user.Sub,
		Username:  user.Username,
		Role:      string(user.Role),
		Action:    action,
		Resource:  r.URL.Path,
		Result:    result,
		Details:   details,
		IPAddress: middleware.ClientIP(r),
		UserAgent: r.Header.Get("User-Agent"),
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		h.auditRepo.Record(ctx, entry)
	}()
}
