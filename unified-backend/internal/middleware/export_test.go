package middleware

// export_test.go — экспортирует приватные функции для тестов в пакете middleware_test.

// WithUser кладёт *domain.User в контекст (обычно делает RequireActiveUser).
import (
	"context"

	"unified-backend/internal/domain"
)

func WithUser(ctx context.Context, u *domain.User) context.Context {
	return context.WithValue(ctx, ctxKeyUser, u)
}

// WithRoles кладёт список ролей в контекст (обычно делает ExtractIdentity).
func WithRoles(ctx context.Context, roles []string) context.Context {
	return context.WithValue(ctx, CtxKeyRoles, roles)
}
