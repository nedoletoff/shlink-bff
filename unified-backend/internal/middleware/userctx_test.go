package middleware_test

import (
	"context"
	"testing"

	"unified-backend/internal/domain"
	"unified-backend/internal/middleware"
)

func TestUserFromCtx_Present(t *testing.T) {
	u := &domain.User{Sub: "sub-1", Role: "editor"}
	ctx := middleware.WithUser(context.Background(), u)
	got := middleware.UserFromCtx(ctx)
	if got == nil || got.Sub != "sub-1" {
		t.Errorf("UserFromCtx: want sub-1, got %v", got)
	}
}

func TestUserFromCtx_Absent(t *testing.T) {
	got := middleware.UserFromCtx(context.Background())
	if got != nil {
		t.Errorf("UserFromCtx: want nil, got %v", got)
	}
}
