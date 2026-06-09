//go:build integration
// +build integration

package handler_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"unified-backend/internal/config"
	"unified-backend/internal/domain"
	"unified-backend/internal/handler"
	"unified-backend/internal/middleware"
	"unified-backend/internal/service"
)

// Сценарий: пользователь с ролью auditor_admin
// (нет short_urls.create) не может создавать короткие ссылки.

// stubPermSvcForIntegration — in-memory реализация PermissionService-подобногоинтерфейса
type stubPermSvcForIntegration struct {
	perms map[string][]string // sub → []actionName
}

func (s *stubPermSvcForIntegration) UserHasPermission(_ context.Context, sub, action string) (bool, error) {
	for _, p := range s.perms[sub] {
		if p == action {
			return true, nil
		}
	}
	return false, nil
}

func (s *stubPermSvcForIntegration) GetUserPermissions(_ context.Context, sub string) ([]string, error) {
	return s.perms[sub], nil
}

func (s *stubPermSvcForIntegration) InvalidateUser(_ string) {}

// stubPermCtrlFromSvc адаптирует стаб в handler.PermChecker
type stubPermCtrlFromSvc struct {
	svc *stubPermSvcForIntegration
}

func (s *stubPermCtrlFromSvc) Check(ctx context.Context, sub string, action string) (bool, error) {
	return s.svc.UserHasPermission(ctx, sub, action)
}

// ── Тесты ─────────────────────────────────────────────────────────────────────────────────────

func makeAuditorUser(sub string) *domain.User {
	return &domain.User{
		ID:   uuid.New(),
		Sub:  sub,
		Role: "auditor_admin",
	}
}

// TestAuditorAdmin_CannotCreateShortURL: auditor_admin без short_urls.create → 403
func TestAuditorAdmin_CannotCreateShortURL(t *testing.T) {
	permSvc := &stubPermSvcForIntegration{
		perms: map[string][]string{
			"auditor-sub": {"users.view", "roles.view", "dashboard.view", "short_urls.view_all"},
			// short_urls.create — нет
		},
	}
	permCtrl := &stubPermCtrlFromSvc{svc: permSvc}

	client := &stubShlinkClient{}
	cfg := &config.Config{ShlinkDefaultDomain: "http://localhost"}
	svc := service.NewShlinkService(client, cfg)
	h := handler.NewShlinkProxyHandler(svc, nil, &stubOwnerRepo{}, cfg, permCtrl)

	body := bytes.NewBufferString(`{"longUrl":"https://example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/shlink/short-urls", body)

	auditor := makeAuditorUser("auditor-sub")
	req = req.WithContext(middleware.WithUser(req.Context(), auditor))

	rec := httptest.NewRecorder()
	h.CreateShortURL(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("want 403 for auditor_admin without short_urls.create, got %d", rec.Code)
	}
}

// TestAuditorAdmin_CanViewUsers: auditor_admin с users.view → GET /api/users = 200
func TestAuditorAdmin_CanViewUsers(t *testing.T) {
	permSvc := &stubPermSvcForIntegration{
		perms: map[string][]string{
			"auditor-sub": {"users.view", "roles.view", "dashboard.view"},
		},
	}
	permCtrl := &stubPermCtrlFromSvc{svc: permSvc}

	// UserHandler с stub репозиториями
	stubUsers := &stubUserRepo{}
	h := handler.NewUserHandler(stubUsers, nil, permCtrl, permSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	auditor := makeAuditorUser("auditor-sub")
	req = req.WithContext(middleware.WithUser(req.Context(), auditor))

	rec := httptest.NewRecorder()
	h.ListUsers(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200 for auditor_admin with users.view, got %d: %s", rec.Code, rec.Body.String())
	}
}
