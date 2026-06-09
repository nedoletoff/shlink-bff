package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"unified-backend/internal/domain"
	"unified-backend/internal/handler"
	"unified-backend/internal/middleware"
)

// ── stubPermissionService ───────────────────────────────────────────────────

// fakePermSvc заменяет *service.PermissionService для тестов через embed.
// Поскольку MeHandler принимает *service.PermissionService (не интерфейс),
// передаём nil и проверяем что permissions = [] (nil-ветка обрабатывается).

func makeUserCtx(r *http.Request, u *domain.User) *http.Request {
	return r.WithContext(middleware.WithUser(r.Context(), u))
}

// ── TestMeHandler_OK ─────────────────────────────────────────────────────────────────

func TestMeHandler_OK(t *testing.T) {
	u := &domain.User{
		ID:       uuid.New(),
		Sub:      "sub-abc",
		Username: "alice",
		Email:    "alice@example.com",
		Role:     "viewer",
		Status:   domain.StatusActive,
	}

	h := handler.NewMeHandler(nil, nil) // permSvc=nil → permissions=[]
	rec := httptest.NewRecorder()
	req := makeUserCtx(httptest.NewRequest(http.MethodGet, "/api/me", nil), u)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	// permissions должен быть [] (не null)
	perms, ok := resp["permissions"]
	if !ok {
		t.Fatal("response missing \"permissions\" field")
	}
	if perms == nil {
		t.Fatal("permissions must not be null")
	}

	// isAdmin не должен присутствовать
	if _, found := resp["isAdmin"]; found {
		t.Fatal("response must not contain \"isAdmin\" field")
	}

	// проверяем основные поля
	if resp["sub"] != u.Sub {
		t.Errorf("want sub=%q, got %q", u.Sub, resp["sub"])
	}
	if resp["username"] != u.Username {
		t.Errorf("want username=%q, got %q", u.Username, resp["username"])
	}
}

func TestMeHandler_Unauthorized(t *testing.T) {
	h := handler.NewMeHandler(nil, nil)
	rec := httptest.NewRecorder()
	// запрос без пользователя в контексте
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/me", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rec.Code)
	}
}

func TestMeHandler_PermissionsNotNull(t *testing.T) {
	u := &domain.User{ID: uuid.New(), Sub: "s", Status: domain.StatusActive}
	h := handler.NewMeHandler(nil, nil)
	rec := httptest.NewRecorder()
	req := makeUserCtx(httptest.NewRequest(http.MethodGet, "/api/me", nil), u)
	h.ServeHTTP(rec, req)

	var resp map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&resp)

	perms := resp["permissions"]
	if perms == nil {
		t.Fatal("permissions must not be null when permSvc=nil")
	}
	// должны быть пустым списком
	slice, ok := perms.([]interface{})
	if !ok {
		t.Fatalf("permissions must be array, got %T", perms)
	}
	if len(slice) != 0 {
		t.Fatalf("want empty permissions, got %v", slice)
	}
}

// заглушаем context.Context чтобы тест компилировался
var _ context.Context = context.Background()
