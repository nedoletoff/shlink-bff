package middleware

import (
	"context"

	"unified-backend/internal/domain"
)

type ctxUserKey struct{}

func WithUser(ctx context.Context, u *domain.User) context.Context {
	return context.WithValue(ctx, ctxUserKey{}, u)
}

func UserFromCtx(ctx context.Context) *domain.User {
	u, _ := ctx.Value(ctxUserKey{}).(*domain.User)
	return u
}
