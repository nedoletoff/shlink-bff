package middleware

// export_test.go — экспортирует приватные символы для тестов в пакете middleware.

import (
	"context"
)

// WithRoles кладёт список ролей в контекст (обычно делает ExtractIdentity).
func WithRoles(ctx context.Context, roles []string) context.Context {
	return context.WithValue(ctx, CtxKeyRoles, roles)
}
