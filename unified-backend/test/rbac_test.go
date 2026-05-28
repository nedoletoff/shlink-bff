package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"unified-backend/internal/domain"
	"unified-backend/internal/middleware"
)

// stubHandler — stub handler, записывает был ли вызван
type stubHandler struct {
	called bool
}

func (h *stubHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.called = true
	w.WriteHeader(http.StatusOK)
}

// buildRequestWithIdentity создаёт http.Request с Identity в контексте
func buildRequestWithIdentity(role string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	ctx := context.WithValue(r.Context(), ctxKey("sub"), "test-sub")
	ctx = context.WithValue(ctx, ctxKey("email"), "test@example.com")
	ctx = context.WithValue(ctx, ctxKey("username"), "testuser")
	ctx = context.WithValue(ctx, ctxKey("role"), role)
	ctx = context.WithValue(ctx, ctxKey("groups"), []string{})
	return r.WithContext(ctx)
}

type ctxKey string

// TestExtractIdentity_MissingHeader — при отсутствии X-Auth-Request-User возвращает 401
func TestExtractIdentity_MissingHeader(t *testing.T) {
	stub := &stubHandler{}
	handler := middleware.ExtractIdentity(stub)

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
	if stub.called {
		t.Error("handler should not have been called")
	}
}

// TestExtractIdentity_WithHeader — при наличии заголовка пропускает дальше
func TestExtractIdentity_WithHeader(t *testing.T) {
	stub := &stubHandler{}
	handler := middleware.ExtractIdentity(stub)

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("X-Auth-Request-User", "sub-123")
	req.Header.Set("X-Auth-Request-Email", "user@example.com")
	req.Header.Set("X-Auth-Request-Preferred-Username", "testuser")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if !stub.called {
		t.Error("handler should have been called with valid identity headers")
	}
}

// TestExtractIdentity_AdminGroup — группа shlink-admins → role=admin
func TestExtractIdentity_AdminGroup(t *testing.T) {
	var capturedIdentity *middleware.Identity
	handler := middleware.ExtractIdentity(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedIdentity = middleware.IdentityFromCtx(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Auth-Request-User", "sub-admin")
	req.Header.Set("X-Auth-Request-Groups", "shlink-admins,developers")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if capturedIdentity == nil {
		t.Fatal("identity not captured")
	}
	if capturedIdentity.Role != "admin" {
		t.Errorf("expected role=admin, got %s", capturedIdentity.Role)
	}
}

// TestExtractIdentity_UserRole — без admin-группы → role=user
func TestExtractIdentity_UserRole(t *testing.T) {
	var capturedRole string
	handler := middleware.ExtractIdentity(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRole = middleware.IdentityFromCtx(r.Context()).Role
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Auth-Request-User", "sub-user")
	req.Header.Set("X-Auth-Request-Groups", "developers,readonly")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if capturedRole != "user" {
		t.Errorf("expected role=user, got %s", capturedRole)
	}
}

// TestUserFromCtx_NilSafe — UserFromCtx при отсутствии user в контексте возвращает nil
func TestUserFromCtx_NilSafe(t *testing.T) {
	ctx := context.Background()
	user := middleware.UserFromCtx(ctx)
	if user != nil {
		t.Error("expected nil user from empty context")
	}
}

// TestWithUser_RoundTrip — WithUser + UserFromCtx корректно хранят данные
func TestWithUser_RoundTrip(t *testing.T) {
	expected := &domain.User{
		Sub:      "test-sub",
		Username: "testuser",
		Role:     domain.RoleAdmin,
		Status:   domain.StatusActive,
	}

	ctx := middleware.WithUser(context.Background(), expected)
	got := middleware.UserFromCtx(ctx)

	if got == nil {
		t.Fatal("user not found in context")
	}
	if got.Sub != expected.Sub {
		t.Errorf("sub: expected %s, got %s", expected.Sub, got.Sub)
	}
	if got.Role != expected.Role {
		t.Errorf("role: expected %s, got %s", expected.Role, got.Role)
	}
}
