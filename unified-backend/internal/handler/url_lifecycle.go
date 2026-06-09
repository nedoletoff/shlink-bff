package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"
	"unified-backend/internal/controller"
	"unified-backend/internal/domain"
	"unified-backend/internal/middleware"
	"unified-backend/internal/service"

	"github.com/go-chi/chi/v5"
)

type URLLifecycleHandler struct {
	shlinkSvc *service.ShlinkService
	ownerRepo OwnershipRepo
	auditRepo AuditRepo
	permCtrl  controller.PermChecker
}

func NewURLLifecycleHandler(
	svc *service.ShlinkService,
	ownerRepo OwnershipRepo,
	auditRepo AuditRepo,
	permCtrl controller.PermChecker,
) *URLLifecycleHandler {
	return &URLLifecycleHandler{
		shlinkSvc: svc,
		ownerRepo: ownerRepo,
		auditRepo: auditRepo,
		permCtrl:  permCtrl,
	}
}

func (h *URLLifecycleHandler) DeactivateURL(w http.ResponseWriter, r *http.Request) {
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
	if err := h.checkDeactivatePermission(r.Context(), user, shortCode); err != nil {
		h.recordAudit(r, user, domain.ActionShortURLDeactivated, "denied", map[string]any{"shortCode": shortCode, "reason": err.Error()})
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}
	patchBody, _ := json.Marshal(map[string]any{"enabled": false})
	if _, err := h.shlinkSvc.Client().UpdateShortURL(r.Context(), user.ShlinkAPIKey, shortCode, bytes.NewReader(patchBody)); err != nil {
		h.recordAudit(r, user, domain.ActionShortURLDeactivated, "error", map[string]any{"shortCode": shortCode, "err": err.Error()})
		writeJSON(w, map[string]string{"error": friendlyShlinkError(err)}, http.StatusBadGateway)
		return
	}
	if err := h.ownerRepo.Deactivate(r.Context(), shortCode, "", user.Sub); err != nil {
		slog.Error("lifecycle: deactivate ownership failed", "err", err)
	}
	h.recordAudit(r, user, domain.ActionShortURLDeactivated, "success", map[string]any{"shortCode": shortCode})
	w.WriteHeader(http.StatusNoContent)
}

func (h *URLLifecycleHandler) ActivateURL(w http.ResponseWriter, r *http.Request) {
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
	if err := h.checkReactivatePermission(r.Context(), user, shortCode); err != nil {
		h.recordAudit(r, user, domain.ActionShortURLActivated, "denied", map[string]any{"shortCode": shortCode, "reason": err.Error()})
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}
	patchBody, _ := json.Marshal(map[string]any{"enabled": true})
	if _, err := h.shlinkSvc.Client().UpdateShortURL(r.Context(), user.ShlinkAPIKey, shortCode, bytes.NewReader(patchBody)); err != nil {
		h.recordAudit(r, user, domain.ActionShortURLActivated, "error", map[string]any{"shortCode": shortCode, "err": err.Error()})
		writeJSON(w, map[string]string{"error": friendlyShlinkError(err)}, http.StatusBadGateway)
		return
	}
	if err := h.ownerRepo.Activate(r.Context(), shortCode, ""); err != nil {
		slog.Error("lifecycle: activate ownership failed", "err", err)
	}
	h.recordAudit(r, user, domain.ActionShortURLActivated, "success", map[string]any{"shortCode": shortCode})
	w.WriteHeader(http.StatusNoContent)
}

func (h *URLLifecycleHandler) DeleteURLPermanently(w http.ResponseWriter, r *http.Request) {
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
	if err := h.checkPermanentDeletePermission(r.Context(), user, shortCode); err != nil {
		h.recordAudit(r, user, domain.ActionShortURLDeletedPermanently, "denied", map[string]any{"shortCode": shortCode, "reason": err.Error()})
		writeJSON(w, map[string]string{"error": "forbidden"}, http.StatusForbidden)
		return
	}
	if err := h.shlinkSvc.Client().DeleteShortURL(r.Context(), user.ShlinkAPIKey, shortCode); err != nil {
		h.recordAudit(r, user, domain.ActionShortURLDeletedPermanently, "error", map[string]any{"shortCode": shortCode, "err": err.Error()})
		writeJSON(w, map[string]string{"error": friendlyShlinkError(err)}, http.StatusBadGateway)
		return
	}
	if err := h.ownerRepo.SoftDelete(r.Context(), shortCode, "", user.Sub); err != nil {
		slog.Error("lifecycle: soft delete failed", "err", err)
	}
	h.recordAudit(r, user, domain.ActionShortURLDeletedPermanently, "success", map[string]any{"shortCode": shortCode})
	w.WriteHeader(http.StatusNoContent)
}

// Методы проверки с использованием новых разрешений
func (h *URLLifecycleHandler) checkDeactivatePermission(ctx context.Context, user *domain.User, shortCode string) error {
	canAll, _ := h.permCtrl.Check(ctx, user.ID, domain.PermShortURLsDeactivateAll)
	if canAll {
		return nil
	}
	canOwn, _ := h.permCtrl.Check(ctx, user.ID, domain.PermShortURLsDeactivateOwn)
	if !canOwn {
		return errors.New("no deactivate permission")
	}
	return h.checkOwnership(ctx, user.Sub, shortCode)
}

func (h *URLLifecycleHandler) checkReactivatePermission(ctx context.Context, user *domain.User, shortCode string) error {
	canAll, _ := h.permCtrl.Check(ctx, user.ID, domain.PermShortURLsReactivateAll)
	if canAll {
		return nil
	}
	canOwn, _ := h.permCtrl.Check(ctx, user.ID, domain.PermShortURLsReactivateOwn)
	if !canOwn {
		return errors.New("no reactivate permission")
	}
	return h.checkOwnership(ctx, user.Sub, shortCode)
}

func (h *URLLifecycleHandler) checkPermanentDeletePermission(ctx context.Context, user *domain.User, shortCode string) error {
	canAll, _ := h.permCtrl.Check(ctx, user.ID, domain.PermShortURLsDeleteAll)
	if canAll {
		return nil
	}
	canOwn, _ := h.permCtrl.Check(ctx, user.ID, domain.PermShortURLsDeleteOwn)
	if !canOwn {
		return errors.New("no delete permission")
	}
	return h.checkOwnership(ctx, user.Sub, shortCode)
}

func (h *URLLifecycleHandler) checkOwnership(ctx context.Context, sub, shortCode string) error {
	isOwner, err := h.ownerRepo.IsOwner(ctx, shortCode, "", sub)
	if err != nil {
		return errors.New("ownership check failed")
	}
	if !isOwner {
		return errors.New("not owner")
	}
	return nil
}

func (h *URLLifecycleHandler) recordAudit(r *http.Request, user *domain.User, action, status string, extra map[string]any) {
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

