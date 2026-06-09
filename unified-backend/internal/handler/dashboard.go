package handler

import (
	"net/http"
	"unified-backend/internal/controller"
	"unified-backend/internal/domain"
	"unified-backend/internal/middleware"
	"unified-backend/internal/service"
)

type DashboardHandler struct {
	shlinkSvc *service.ShlinkService
	ownerRepo service.StatsOwnershipRepo
	permCtrl  controller.PermChecker
}

func NewDashboardHandler(shlinkSvc *service.ShlinkService, ownerRepo service.StatsOwnershipRepo, permCtrl controller.PermChecker) *DashboardHandler {
	return &DashboardHandler{
		shlinkSvc: shlinkSvc,
		ownerRepo: ownerRepo,
		permCtrl:  permCtrl,
	}
}

func (h *DashboardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromCtx(r.Context())
	if user == nil {
		writeJSON(w, map[string]string{"error": "unauthorized"}, http.StatusUnauthorized)
		return
	}
	canViewAll, _ := h.permCtrl.Check(r.Context(), user.ID, domain.PermShortURLsViewAll)
	stats, err := h.shlinkSvc.GetDashboardStats(r.Context(), user, canViewAll, h.ownerRepo)
	if err != nil {
		writeJSON(w, map[string]string{"error": err.Error()}, http.StatusInternalServerError)
		return
	}
	perms, _ := h.permCtrl.GetUserPermissions(r.Context(), user.Sub)
	writeJSON(w, map[string]interface{}{
		"stats":       stats,
		"permissions": perms,
	}, http.StatusOK)
}

