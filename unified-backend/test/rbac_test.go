package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
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

// --- Вспомогательная функция: подаёт запрос, возвращает role
func captureRole(groups string) string {
	var id *middleware.Identity
	handler := middleware.ExtractIdentity(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id = middleware.IdentityFromCtx(r.Context())
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Auth-Request-User", "sub-test")
	if groups != "" {
		req.Header.Set("X-Auth-Request-Groups", groups)
	}
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if id == nil {
		return ""
	}
	return id.Role
}

// --- Дефолтные группы ---

func TestExtractIdentity_AdminGroup_Default(t *testing.T) {
	os.Unsetenv("ADMIN_GROUPS")
	middleware.ReloadAdminGroups()
	if role := captureRole("shlink-admins,developers"); role != "admin" {
		t.Errorf("expected admin, got %s", role)
	}
}

func TestExtractIdentity_AdminGroup_LegacyAdmin(t *testing.T) {
	os.Unsetenv("ADMIN_GROUPS")
	middleware.ReloadAdminGroups()
	if role := captureRole("admin"); role != "admin" {
		t.Errorf("expected admin (legacy group), got %s", role)
	}
}

func TestExtractIdentity_UserRole_NoAdminGroup(t *testing.T) {
	os.Unsetenv("ADMIN_GROUPS")
	middleware.ReloadAdminGroups()
	if role := captureRole("developers,readonly"); role != "user" {
		t.Errorf("expected user, got %s", role)
	}
}

func TestExtractIdentity_UserRole_EmptyGroups(t *testing.T) {
	os.Unsetenv("ADMIN_GROUPS")
	middleware.ReloadAdminGroups()
	if role := captureRole(""); role != "user" {
		t.Errorf("expected user for empty groups, got %s", role)
	}
}

// --- Кастомные группы через ADMIN_GROUPS ---

func TestExtractIdentity_CustomAdminGroup(t *testing.T) {
	os.Setenv("ADMIN_GROUPS", "shadmin,superusers")
	middleware.ReloadAdminGroups()
	defer func() {
		os.Unsetenv("ADMIN_GROUPS")
		middleware.ReloadAdminGroups()
	}()

	tests := []struct {
		groups string
		want   string
	}{
		{"shadmin", "admin"},
		{"superusers,devs", "admin"},
		{"shlink-admins", "user"}, // старое имя больше не работает с кастомным ADMIN_GROUPS
		{"admin", "user"},          // аналогично
		{"developers", "user"},
	}
	for _, tc := range tests {
		got := captureRole(tc.groups)
		if got != tc.want {
			t.Errorf("groups=%q: expected %s, got %s", tc.groups, tc.want, got)
		}
	}
}

func TestExtractIdentity_CaseInsensitive(t *testing.T) {
	os.Setenv("ADMIN_GROUPS", "Shlink-Admins")
	middleware.ReloadAdminGroups()
	defer func() {
		os.Unsetenv("ADMIN_GROUPS")
		middleware.ReloadAdminGroups()
	}()
	if role := captureRole("SHLINK-ADMINS"); role != "admin" {
		t.Errorf("expected admin (case-insensitive), got %s", role)
	}
}

// --- Заполнение полей Identity ---

func TestExtractIdentity_FieldsPopulated(t *testing.T) {
	os.Unsetenv("ADMIN_GROUPS")
	middleware.ReloadAdminGroups()

	var id *middleware.Identity
	handler := middleware.ExtractIdentity(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id = middleware.IdentityFromCtx(r.Context())
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Auth-Request-User", "sub-abc")
	req.Header.Set("X-Auth-Request-Email", "test@example.local")
	req.Header.Set("X-Auth-Request-Preferred-Username", "testuser")
	req.Header.Set("X-Auth-Request-Groups", "shlink-admins")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if id.Sub != "sub-abc" {
		t.Errorf("Sub: got %q", id.Sub)
	}
	if id.Email != "test@example.local" {
		t.Errorf("Email: got %q", id.Email)
	}
	if id.Username != "testuser" {
		t.Errorf("Username: got %q", id.Username)
	}
	if id.Role != "admin" {
		t.Errorf("Role: got %q", id.Role)
	}
	if len(id.Groups) != 1 || id.Groups[0] != "shlink-admins" {
		t.Errorf("Groups: got %v", id.Groups)
	}
}

// --- context helpers ---

func TestUserFromCtx_NilSafe(t *testing.T) {
	ctx := context.Background()
	user := middleware.UserFromCtx(ctx)
	if user != nil {
		t.Error("expected nil user from empty context")
	}
}

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
		t.Errorf("Sub: expected %s, got %s", expected.Sub, got.Sub)
	}
	if got.Role != expected.Role {
		t.Errorf("Role: expected %s, got %s", expected.Role, got.Role)
	}
}
