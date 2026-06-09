package controller

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"unified-backend/internal/middleware"
)

// PermChecker — минимальный интерфейс для stubов в тестах.
type PermChecker interface {
	Check(ctx context.Context, userID uuid.UUID, action string) (bool, error)
	Authorize(action string) func(http.Handler) http.Handler
}

// PermissionSvc — подмножество service.PermissionService без циклической зависимости.
type PermissionSvc interface {
	UserHasPermission(ctx context.Context, userID uuid.UUID, action string) (bool, error)
}

// PermissionController реализует PermChecker.
type PermissionController struct {
	svc PermissionSvc
}

func NewPermissionController(svc PermissionSvc) *PermissionController {
	return &PermissionController{svc: svc}
}

// Check проверяет, имеет ли пользователь из контекста нужное разрешение.
func (c *PermissionController) Check(ctx context.Context, userID uuid.UUID, action string) (bool, error) {
	return c.svc.UserHasPermission(ctx, userID, action)
}

// Authorize — chi-совместимое middleware: если у текущего пользователя нет action — возвращает 403.
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

// Compile-time check.
var _ PermChecker = (*PermissionController)(nil)
