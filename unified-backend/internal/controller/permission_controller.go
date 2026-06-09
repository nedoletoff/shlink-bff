package controller

import (
	"context"
	"net/http"
	"unified-backend/internal/middleware"

	"github.com/google/uuid"
)

type PermissionSvc interface {
	UserHasPermission(ctx context.Context, userID uuid.UUID, action string) (bool, error)
	UserHasPermissionBySub(ctx context.Context, sub string, action string) (bool, error)
	GetUserPermissions(ctx context.Context, sub string) ([]string, error) // добавлено
}

type PermChecker interface {
	Check(ctx context.Context, userID uuid.UUID, action string) (bool, error)
	CheckSub(ctx context.Context, sub string, action string) (bool, error)
	GetUserPermissions(ctx context.Context, sub string) ([]string, error) // добавлено
	Authorize(action string) func(http.Handler) http.Handler
}

type PermissionController struct {
	svc PermissionSvc
}

func NewPermissionController(svc PermissionSvc) *PermissionController {
	return &PermissionController{svc: svc}
}

func (c *PermissionController) Check(ctx context.Context, userID uuid.UUID, action string) (bool, error) {
	return c.svc.UserHasPermission(ctx, userID, action)
}

func (c *PermissionController) CheckSub(ctx context.Context, sub string, action string) (bool, error) {
	return c.svc.UserHasPermissionBySub(ctx, sub, action)
}

func (c *PermissionController) GetUserPermissions(ctx context.Context, sub string) ([]string, error) {
	return c.svc.GetUserPermissions(ctx, sub)
}

func (c *PermissionController) Authorize(action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := middleware.UserFromCtx(r.Context())
			if user == nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			ok, err := c.svc.UserHasPermission(r.Context(), user.ID, action)
			if err != nil {
				http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
				return
			}
			if !ok {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

