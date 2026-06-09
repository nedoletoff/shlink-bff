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
	"unified-backend/internal/config"
	"unified-backend/internal/controller"
	"unified-backend/internal/domain"
	"unified-backend/internal/middleware"
	"unified-backend/internal/service"
	"unified-backend/internal/shlink"

	"github.com/go-chi/chi/v5"
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

// ─────────────────────────────────────────────────────────────────────────────
//  Request/response structures
// ─────────────────────────────────────────────────────────────────────────────

type createShortURLRequest struct {
	LongURL    string   `json:"longUrl"`
	Title      string   `json:"title"`
	CustomSlug string   `json:"customSlug"`
	Domain     string   `json:"domain"`
	Tags       []string `json:"tags"`
	MaxVisits  *int     `json:"maxVisits"`
	ValidSince *string  `json:"validSince"`
	ValidUntil *string  `json:"validUntil"`
}

type updateShortURLRequest struct {
	LongURL    *string  `json:"longUrl"`
	Title      *string  `json:"title"`
	CustomSlug *string  `json:"customSlug"`
	Domain     *string  `json:"domain"`
	Tags       []string `json:"tags"`
	MaxVisits  *int     `json:"maxVisits"`
	ValidSince *string  `json:"validSince"`
	ValidUntil *string  `json:"validUntil"`
	Enabled    *bool    `json:"enabled"`
}

// ─────────────────────────────────────────────────────────────────────────────
//  Helper functions
// ─────────────────────────────────────────────────────────────────────────────

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

func parseTimePtr(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil
	}
	return &t
}

func derefInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
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

// ─────────────────────────────────────────────────────────────────────────────
//  LIST SHORT URLS
// ─────────────────────────────────────────────────────────────────────────────

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
	// даже если нет view_all, но есть create – всё равно можно видеть свои
	if !canViewAll {
		hasCreate, _ := h.permCtrl.Check(r.Context(), user.ID, domain.PermShortURLsCreateOwn)
		if !hasCreate && !canViewAll {
			h.recordAudit(r, user, "list_short_urls", "denied", map[string]any{"reason": "no permission"})
			writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
			return
		}
	}

	statusFilter := r.URL.Query().Get("status")
	if statusFilter == "" {
		statusFilter = "active"
	}

	resp, err := h.shlinkSvc.Client().GetShortURLs(r.Context(), user.ShlinkAPIKey, r.URL.RawQuery)
	if err != nil {
		slog.Error("proxy: get short-urls failed", "err", err)
		h.recordAudit(r, user, "list_short_urls", "error", map[string]any{"err": err.Error()})
		writeJSON(w, map[string]string{"error": "shlink unavailable"}, http.StatusBadGateway)
		return
	}

	// Если нет прав на все – фильтруем по владельцу
	if !canViewAll {
		ownedCodes, _ := h.ownerRepo.GetShortCodeSet(r.Context(), user.Sub)
		filtered := make([]shlink.ShortURL, 0, len(resp.ShortURLs.Data))
		for _, u := range resp.ShortURLs.Data {
			if _, ok := ownedCodes[u.ShortCode]; ok {
				filtered = append(filtered, u)
			}
		}
		resp.ShortURLs.Data = filtered
		resp.ShortURLs.Pagination.TotalItems = len(filtered)
		resp.ShortURLs.Pagination.ItemsInCurrentPage = len(filtered)
		resp.ShortURLs.Pagination.PagesCount = 1
		resp.ShortURLs.Pagination.CurrentPage = 1
	}

	// Фильтр по статусу (активна/деактивирована)
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

	// Обогащаем метаданными из url_ownership
	if len(resp.ShortURLs.Data) > 0 {
		codes := make([]string, len(resp.ShortURLs.Data))
		for i, u := range resp.ShortURLs.Data {
			codes[i] = u.ShortCode
		}
		batch, _ := h.ownerRepo.GetBatch(r.Context(), codes, "")
		for i := range resp.ShortURLs.Data {
			if meta, ok := batch[resp.ShortURLs.Data[i].ShortCode]; ok {
				resp.ShortURLs.Data[i].Title = meta.Title
				if meta.ValidSince != nil {
					s := meta.ValidSince.Format(time.RFC3339)
					resp.ShortURLs.Data[i].ValidSince = &s
				}
				if meta.ValidUntil != nil {
					s := meta.ValidUntil.Format(time.RFC3339)
					resp.ShortURLs.Data[i].ValidUntil = &s
				}
				if meta.MaxVisits > 0 {
					resp.ShortURLs.Data[i].MaxVisits = &meta.MaxVisits
				}
				resp.ShortURLs.Data[i].Enabled = meta.IsActive
			}
		}
	}

	h.recordAudit(r, user, "list_short_urls", "success", map[string]any{"status": statusFilter})
	writeJSON(w, resp, http.StatusOK)
}

// ─────────────────────────────────────────────────────────────────────────────
//  CREATE SHORT URL
// ─────────────────────────────────────────────────────────────────────────────

func (h *ShlinkProxyHandler) CreateShortURL(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}

	// Проверка права на создание (свои или любые)
	canCreateOwn, _ := h.permCtrl.Check(r.Context(), user.ID, domain.PermShortURLsCreateOwn)
	canCreateAll, _ := h.permCtrl.Check(r.Context(), user.ID, domain.PermShortURLsCreate)
	if !canCreateOwn && !canCreateAll {
		h.recordAudit(r, user, "create_short_url", "denied", map[string]any{"reason": "no create permission"})
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

	// Проверка дополнительных параметров
	if req.CustomSlug != "" {
		ok, _ := h.permCtrl.Check(r.Context(), user.ID, domain.PermShortURLsCustomSlug)
		if !ok {
			writeJSON(w, map[string]string{"error": "custom slug not allowed"}, http.StatusForbidden)
			return
		}
	}
	if req.ValidSince != nil || req.ValidUntil != nil {
		ok, _ := h.permCtrl.Check(r.Context(), user.ID, domain.PermShortURLsTimeLimits)
		if !ok {
			writeJSON(w, map[string]string{"error": "time limits not allowed"}, http.StatusForbidden)
			return
		}
	}
	if req.MaxVisits != nil && *req.MaxVisits > 0 {
		ok, _ := h.permCtrl.Check(r.Context(), user.ID, domain.PermShortURLsVisitLimits)
		if !ok {
			writeJSON(w, map[string]string{"error": "visit limits not allowed"}, http.StatusForbidden)
			return
		}
	}

	// Ограничение по домену
	if err := h.shlinkSvc.EnforceDomain(r.Context(), user, req.Domain); err != nil {
		h.recordAudit(r, user, "create_short_url", "denied", map[string]any{"reason": err.Error(), "domain": req.Domain})
		writeJSON(w, map[string]string{"error": err.Error()}, http.StatusForbidden)
		return
	}

	// Обработка префикса slug
	slugPtr := &req.CustomSlug
	if req.CustomSlug == "" {
		slugPtr = nil
	}
	enforced, err := h.shlinkSvc.EnforceSlugPrefix(r.Context(), user, slugPtr)
	if err != nil {
		h.recordAudit(r, user, "create_short_url", "denied", map[string]any{"reason": err.Error()})
		writeJSON(w, map[string]string{"error": err.Error()}, http.StatusForbidden)
		return
	}
	payload := make(map[string]any)
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		writeJSON(w, map[string]string{"error": "invalid json"}, http.StatusBadRequest)
		return
	}
	if enforced != "" {
		payload["customSlug"] = enforced
		req.CustomSlug = enforced
	}
	modifiedBody, _ := json.Marshal(payload)

	// Вызов Shlink
	result, err := h.shlinkSvc.Client().CreateShortURL(r.Context(), user.ShlinkAPIKey, bytes.NewReader(modifiedBody))
	if err != nil {
		slog.Error("proxy: create short-url failed", "err", err)
		h.recordAudit(r, user, "create_short_url", "error", map[string]any{"err": err.Error()})
		writeJSON(w, map[string]string{"error": friendlyShlinkError(err)}, http.StatusBadRequest)
		return
	}

	// Сохраняем метаданные в БД
	metadata := &domain.ShortURLMetadata{
		ShortCode:  result.ShortCode,
		Title:      req.Title,
		IsActive:   true,
		ValidSince: parseTimePtr(req.ValidSince),
		ValidUntil: parseTimePtr(req.ValidUntil),
		MaxVisits:  derefInt(req.MaxVisits),
		IsPublic:   false,
		Tags:       req.Tags,
	}
	if err := h.ownerRepo.Save(r.Context(), result.ShortCode, user.Sub, user.Username, req.Domain, metadata); err != nil {
		slog.Error("proxy: save metadata failed", "err", err)
	}

	h.recordAudit(r, user, "create_short_url", "success", map[string]any{"shortCode": result.ShortCode})
	writeJSON(w, result, http.StatusCreated)
}

// ─────────────────────────────────────────────────────────────────────────────
//  UPDATE SHORT URL
// ─────────────────────────────────────────────────────────────────────────────

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

	// Проверка прав (update.all или update.own + владение)
	if err := h.checkModifyPermission(r.Context(), user, shortCode, false); err != nil {
		h.recordAudit(r, user, "update_short_url", "denied", map[string]any{"shortCode": shortCode, "reason": err.Error()})
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSON(w, map[string]string{"error": "bad request"}, http.StatusBadRequest)
		return
	}
	var req updateShortURLRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		writeJSON(w, map[string]string{"error": "invalid json"}, http.StatusBadRequest)
		return
	}

	// Проверка дополнительных параметров (если они переданы в обновлении)
	if req.CustomSlug != nil && *req.CustomSlug != "" {
		ok, _ := h.permCtrl.Check(r.Context(), user.ID, domain.PermShortURLsCustomSlug)
		if !ok {
			writeJSON(w, map[string]string{"error": "custom slug not allowed"}, http.StatusForbidden)
			return
		}
	}
	if (req.ValidSince != nil && *req.ValidSince != "") || (req.ValidUntil != nil && *req.ValidUntil != "") {
		ok, _ := h.permCtrl.Check(r.Context(), user.ID, domain.PermShortURLsTimeLimits)
		if !ok {
			writeJSON(w, map[string]string{"error": "time limits not allowed"}, http.StatusForbidden)
			return
		}
	}
	if req.MaxVisits != nil && *req.MaxVisits > 0 {
		ok, _ := h.permCtrl.Check(r.Context(), user.ID, domain.PermShortURLsVisitLimits)
		if !ok {
			writeJSON(w, map[string]string{"error": "visit limits not allowed"}, http.StatusForbidden)
			return
		}
	}

	// Обновляем в Shlink
	result, err := h.shlinkSvc.Client().UpdateShortURL(r.Context(), user.ShlinkAPIKey, shortCode, bytes.NewReader(bodyBytes))
	if err != nil {
		slog.Error("proxy: update failed", "err", err)
		h.recordAudit(r, user, "update_short_url", "error", map[string]any{"shortCode": shortCode, "err": err.Error()})
		writeJSON(w, map[string]string{"error": friendlyShlinkError(err)}, http.StatusBadRequest)
		return
	}

	// Обновляем метаданные в БД (если есть изменения в title, validSince, validUntil, maxVisits, tags)
	meta, _ := h.ownerRepo.GetOwnership(r.Context(), shortCode, "")
	if meta != nil {
		updateMeta := false
		if req.Title != nil && *req.Title != meta.Title {
			meta.Title = *req.Title
			updateMeta = true
		}
		if req.ValidSince != nil {
			meta.ValidSince = parseTimePtr(req.ValidSince)
			updateMeta = true
		}
		if req.ValidUntil != nil {
			meta.ValidUntil = parseTimePtr(req.ValidUntil)
			updateMeta = true
		}
		if req.MaxVisits != nil {
			meta.MaxVisits = *req.MaxVisits
			updateMeta = true
		}
		if req.Tags != nil {
			meta.Tags = req.Tags
			updateMeta = true
		}
		if updateMeta {
			// Здесь нужен метод UpdateMetadata в репозитории
			// Для простоты пересохраним (но лучше обновлять конкретные поля)
			_ = h.ownerRepo.Save(r.Context(), shortCode, user.Sub, user.Username, "", &domain.ShortURLMetadata{
				ShortCode:  shortCode,
				Title:      meta.Title,
				IsActive:   meta.IsActive,
				ValidSince: meta.ValidSince,
				ValidUntil: meta.ValidUntil,
				MaxVisits:  meta.MaxVisits,
				Tags:       meta.Tags,
			})
		}
	}

	h.recordAudit(r, user, "update_short_url", "success", map[string]any{"shortCode": shortCode})
	writeJSON(w, result, http.StatusOK)
}

// ─────────────────────────────────────────────────────────────────────────────
//  DELETE SHORT URL (hard delete)
// ─────────────────────────────────────────────────────────────────────────────

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
	// Проверка прав (delete.all или delete.own + владение)
	if err := h.checkModifyPermission(r.Context(), user, shortCode, true); err != nil {
		h.recordAudit(r, user, "delete_short_url", "denied", map[string]any{"shortCode": shortCode, "reason": err.Error()})
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}

	if err := h.shlinkSvc.Client().DeleteShortURL(r.Context(), user.ShlinkAPIKey, shortCode); err != nil {
		slog.Error("proxy: delete failed", "err", err)
		h.recordAudit(r, user, "delete_short_url", "error", map[string]any{"shortCode": shortCode, "err": err.Error()})
		writeJSON(w, map[string]string{"error": friendlyShlinkError(err)}, http.StatusBadGateway)
		return
	}
	// Hard delete из БД
	if err := h.ownerRepo.HardDelete(r.Context(), shortCode, ""); err != nil {
		slog.Error("proxy: hard delete ownership failed", "err", err)
	}
	h.recordAudit(r, user, domain.ActionShortURLDeletedPermanently, "success", map[string]any{"shortCode": shortCode})
	w.WriteHeader(http.StatusNoContent)
}

// ─────────────────────────────────────────────────────────────────────────────
//  CHECK PERMISSION FOR MODIFICATION (update / delete)
// ─────────────────────────────────────────────────────────────────────────────

func (h *ShlinkProxyHandler) checkModifyPermission(ctx context.Context, user *domain.User, shortCode string, isDelete bool) error {
	var permAll, permOwn string
	if isDelete {
		permAll = domain.PermShortURLsDeleteAll
		permOwn = domain.PermShortURLsDeleteOwn
	} else {
		permAll = domain.PermShortURLsUpdateAll
		permOwn = domain.PermShortURLsUpdateOwn
	}
	canAll, err := h.permCtrl.Check(ctx, user.ID, permAll)
	if err != nil {
		return err
	}
	if canAll {
		return nil
	}
	canOwn, err := h.permCtrl.Check(ctx, user.ID, permOwn)
	if err != nil {
		return err
	}
	if !canOwn {
		return errors.New("no permission")
	}
	// Проверка владения
	isOwner, err := h.ownerRepo.IsOwner(ctx, shortCode, "", user.Sub)
	if err != nil {
		return err
	}
	if !isOwner {
		return errors.New("not owner")
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
//  TAGS endpoints (список, создание, переименование, удаление)
// ─────────────────────────────────────────────────────────────────────────────

func (h *ShlinkProxyHandler) ListTags(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}
	resp, err := h.shlinkSvc.Client().GetTags(r.Context(), user.ShlinkAPIKey)
	if err != nil {
		writeJSON(w, map[string]string{"error": "shlink unavailable"}, http.StatusBadGateway)
		return
	}
	writeJSON(w, resp, http.StatusOK)
}

func (h *ShlinkProxyHandler) CreateTag(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}
	// Для создания тега требуется право manage_tags.own или .all (поскольку теги привязаны к ссылкам)
	ok, _ := h.permCtrl.Check(r.Context(), user.ID, domain.PermShortURLsManageTagsOwn)
	if !ok {
		ok, _ = h.permCtrl.Check(r.Context(), user.ID, domain.PermShortURLsManageTagsAll)
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
		writeJSON(w, map[string]string{"error": err.Error()}, http.StatusBadGateway)
		return
	}
	writeJSON(w, resp, http.StatusCreated)
}

func (h *ShlinkProxyHandler) RenameTag(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}
	ok, _ := h.permCtrl.Check(r.Context(), user.ID, domain.PermShortURLsManageTagsAll)
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
		writeJSON(w, map[string]string{"error": err.Error()}, http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ShlinkProxyHandler) DeleteTag(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}
	ok, _ := h.permCtrl.Check(r.Context(), user.ID, domain.PermShortURLsManageTagsAll)
	if !ok {
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}
	tagID := chi.URLParam(r, "tagId")
	if tagID == "" {
		writeJSON(w, map[string]string{"error": "tagId required"}, http.StatusBadRequest)
		return
	}
	if err := h.shlinkSvc.Client().DeleteTags(r.Context(), user.ShlinkAPIKey, []string{tagID}); err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

