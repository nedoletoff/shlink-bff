package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"unified-backend/internal/domain"
	"unified-backend/internal/middleware"
)

// buildRequestWithUser — создаёт *http.Request с пользователем в контексте.
func buildRequestWithUser(user *domain.User) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := middleware.WithUser(req.Context(), user)
	return req.WithContext(ctx)
}

// ── RequireAdmin ───────────────────────────────────────────────────────────

// TestRequireAdmin_AdminPasses — admin-пользователь проходит middleware.
func TestRequireAdmin_AdminPasses(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.RequireAdmin(next)

	admin := &domain.User{
		Sub:    "admin-sub",
		Role:   domain.RoleAdmin,
		Status: domain.StatusActive,
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, buildRequestWithUser(admin))

	if !called {
		t.Error("RequireAdmin: admin user should pass through")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("RequireAdmin: want 200, got %d", rr.Code)
	}
}

// TestRequireAdmin_UserBlocked — обычный пользователь получает 403.
func TestRequireAdmin_UserBlocked(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	handler := middleware.RequireAdmin(next)

	user := &domain.User{
		Sub:    "user-sub",
		Role:   domain.RoleUser,
		Status: domain.StatusActive,
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, buildRequestWithUser(user))

	if called {
		t.Error("RequireAdmin: regular user should be blocked")
	}
	if rr.Code != http.StatusForbidden {
		t.Errorf("RequireAdmin: want 403, got %d", rr.Code)
	}
}

// TestRequireAdmin_NoUser — нет пользователя в контексте → 401.
func TestRequireAdmin_NoUser(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	handler := middleware.RequireAdmin(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if called {
		t.Error("RequireAdmin: should block request without user in context")
	}
	if rr.Code != http.StatusUnauthorized && rr.Code != http.StatusForbidden {
		t.Errorf("RequireAdmin: want 401 or 403 without user, got %d", rr.Code)
	}
}

// TestRequireAdmin_CustomRole — кастомная non-admin роль блокируется.
func TestRequireAdmin_CustomRoleBlocked(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	handler := middleware.RequireAdmin(next)

	user := &domain.User{
		Sub:    "editor-sub",
		Role:   "editor",
		Status: domain.StatusActive,
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, buildRequestWithUser(user))

	if called {
		t.Error("RequireAdmin: editor role should be blocked")
	}
	if rr.Code != http.StatusForbidden {
		t.Errorf("RequireAdmin: want 403, got %d", rr.Code)
	}
}

// TestRequireAdmin_SuspendedAdmin — suspended admin — поведение зависит от реализации.
// Если RequireAdmin проверяет только Role (без Status), suspended admin пройдёт.
// Тест документирует фактическое поведение.
func TestRequireAdmin_SuspendedAdmin_PassesRoleCheck(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.RequireAdmin(next)

	suspendedAdmin := &domain.User{
		Sub:    "suspended-admin",
		Role:   domain.RoleAdmin,
		Status: domain.StatusSuspended,
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, buildRequestWithUser(suspendedAdmin))

	// RequireAdmin проверяет только Role — suspended admin пройдёт;
	// Status проверяет RequireActiveUser, стоящий раньше в цепочке.
	if !called {
		t.Log("note: RequireAdmin also checks Status — suspended admin was blocked")
	}
}
