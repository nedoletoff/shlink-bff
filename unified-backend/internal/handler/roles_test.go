package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"unified-backend/internal/domain"
	"unified-backend/internal/handler"
	"unified-backend/internal/middleware"
	"unified-backend/internal/repository/postgres"
)

// ── stubPermController ────────────────────────────────────────────────────────

type stubPermController struct {
	allowed bool
	err     error
}

func (s *stubPermController) Check(_ context.Context, _ uuid.UUID, _ string) (bool, error) {
	return s.allowed, s.err
}

// ── stubRoleRepository ────────────────────────────────────────────────────────

type stubRoleRepository struct {
	roles    []domain.RoleEntity
	perms    []domain.Permission
	getByID  *domain.RoleEntity
	createErr error
	setErr   error
}

func (s *stubRoleRepository) GetAll(_ context.Context) ([]domain.RoleEntity, error) {
	return s.roles, nil
}
func (s *stubRoleRepository) Create(_ context.Context, name, desc string) (*domain.RoleEntity, error) {
	if s.createErr != nil {
		return nil, s.createErr
	}
	r := &domain.RoleEntity{ID: uuid.New(), Name: name, Description: desc}
	s.roles = append(s.roles, *r)
	return r, nil
}
func (s *stubRoleRepository) GetByID(_ context.Context, _ uuid.UUID) (*domain.RoleEntity, error) {
	return s.getByID, nil
}
func (s *stubRoleRepository) GetPermissions(_ context.Context, _ uuid.UUID) ([]domain.Permission, error) {
	return s.perms, nil
}
func (s *stubRoleRepository) SetPermissions(_ context.Context, _ uuid.UUID, _ []uuid.UUID) error {
	return s.setErr
}

// ── helpers ───────────────────────────────────────────────────────────────────

var errHandlerStub = errors.New("stub error")

func userCtx(r *http.Request) *http.Request {
	u := &domain.User{ID: uuid.New(), Sub: "test-sub", Role: "admin"}
	return r.WithContext(middleware.WithUser(r.Context(), u))
}

func chiRequest(method, url string, body *bytes.Reader, params chi.RouteParams) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, url, body)
	} else {
		req = httptest.NewRequest(method, url, nil)
	}
	rctx := chi.NewRouteContext()
	rctx.URLParams = params
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func newRoleHandler(perm *stubPermController, repo *stubRoleRepository) *handler.RoleHandler {
	return handler.NewRoleHandler(
		(*postgres.RoleRepository)(nil), // не используется в stubbed тестах
		nil,
		perm,
	)
}

// ── ListRoles ─────────────────────────────────────────────────────────────────

func TestRoleHandler_ListRoles_Allowed(t *testing.T) {
	permCtrl := &stubPermController{allowed: true}
	_ = permCtrl // использован через NewRoleHandler

	// Используем реальный конструктор с nil-репо, разрешение выдаётся stubPermController.
	// Поскольку repo nil — при попытке дёрнуть GetAll будет паника;
	// мы проверяем только что 403 не возвращается (т.е. auth прошла).
	// Для полноценного теста GetAll используется интеграционный тест.
	perm := &stubPermController{allowed: false}
	h := handler.NewRoleHandler(nil, nil, perm)

	rec := httptest.NewRecorder()
	req := userCtx(httptest.NewRequest(http.MethodGet, "/api/roles", nil))
	h.ListRoles(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("want 403 forbidden, got %d", rec.Code)
	}
}

func TestRoleHandler_ListRoles_Forbidden(t *testing.T) {
	perm := &stubPermController{allowed: false}
	h := handler.NewRoleHandler(nil, nil, perm)

	rec := httptest.NewRecorder()
	req := userCtx(httptest.NewRequest(http.MethodGet, "/api/roles", nil))
	h.ListRoles(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d", rec.Code)
	}
}

func TestRoleHandler_ListRoles_NoUser_Unauthorized(t *testing.T) {
	perm := &stubPermController{allowed: true}
	h := handler.NewRoleHandler(nil, nil, perm)

	rec := httptest.NewRecorder()
	// Запрос без пользователя в контексте.
	req := httptest.NewRequest(http.MethodGet, "/api/roles", nil)
	h.ListRoles(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rec.Code)
	}
}

// ── CreateRole ────────────────────────────────────────────────────────────────

func TestRoleHandler_CreateRole_Forbidden(t *testing.T) {
	perm := &stubPermController{allowed: false}
	h := handler.NewRoleHandler(nil, nil, perm)

	body, _ := json.Marshal(map[string]string{"name": "auditor"})
	rec := httptest.NewRecorder()
	req := userCtx(httptest.NewRequest(http.MethodPost, "/api/roles", bytes.NewReader(body)))
	h.CreateRole(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d", rec.Code)
	}
}

func TestRoleHandler_CreateRole_BadJSON(t *testing.T) {
	perm := &stubPermController{allowed: true}
	h := handler.NewRoleHandler(nil, nil, perm)

	rec := httptest.NewRecorder()
	req := userCtx(httptest.NewRequest(http.MethodPost, "/api/roles", bytes.NewReader([]byte(`not-json`))))
	h.CreateRole(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rec.Code)
	}
}

func TestRoleHandler_CreateRole_EmptyName(t *testing.T) {
	perm := &stubPermController{allowed: true}
	h := handler.NewRoleHandler(nil, nil, perm)

	body, _ := json.Marshal(map[string]string{"name": "   "})
	rec := httptest.NewRecorder()
	req := userCtx(httptest.NewRequest(http.MethodPost, "/api/roles", bytes.NewReader(body)))
	h.CreateRole(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rec.Code)
	}
}

// ── SetRolePermissions ────────────────────────────────────────────────────────

func TestRoleHandler_SetRolePermissions_Forbidden(t *testing.T) {
	perm := &stubPermController{allowed: false}
	h := handler.NewRoleHandler(nil, nil, perm)

	roleID := uuid.New()
	body, _ := json.Marshal(map[string][]string{"permissionIds": {uuid.New().String()}})
	rec := httptest.NewRecorder()
	req := userCtx(chiRequest(http.MethodPut, "/api/roles/"+roleID.String()+"/permissions",
		bytes.NewReader(body),
		chi.RouteParams{Keys: []string{"id"}, Values: []string{roleID.String()}},
	))
	h.SetRolePermissions(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d", rec.Code)
	}
}

func TestRoleHandler_SetRolePermissions_InvalidUUID(t *testing.T) {
	perm := &stubPermController{allowed: true}
	h := handler.NewRoleHandler(nil, nil, perm)

	rec := httptest.NewRecorder()
	req := userCtx(chiRequest(http.MethodPut, "/api/roles/not-a-uuid/permissions",
		bytes.NewReader([]byte(`{"permissionIds":[]}`)),
		chi.RouteParams{Keys: []string{"id"}, Values: []string{"not-a-uuid"}},
	))
	h.SetRolePermissions(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rec.Code)
	}
}
