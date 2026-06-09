package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"unified-backend/internal/config"
	"unified-backend/internal/domain"
	"unified-backend/internal/handler"
	"unified-backend/internal/middleware"
	"unified-backend/internal/repository/postgres"
	"unified-backend/internal/service"
	"unified-backend/internal/shlink"
)

// ── Stubs ───────────────────────────────────────────────────────────────────────────────────────

type stubOwnerRepo struct {
	isOwner    bool
	ownerError error
}

func (s *stubOwnerRepo) Save(_ context.Context, _, _, _, _ string) error { return nil }
func (s *stubOwnerRepo) IsOwner(_ context.Context, _, _, _ string) (bool, error) {
	return s.isOwner, s.ownerError
}
func (s *stubOwnerRepo) HardDelete(_ context.Context, _, _ string) error        { return nil }
func (s *stubOwnerRepo) SetActive(_ context.Context, _, _ string, _ bool) error { return nil }
func (s *stubOwnerRepo) GetShortCodeSet(_ context.Context, _ string) (map[string]struct{}, error) {
	return map[string]struct{}{}, nil
}
func (s *stubOwnerRepo) GetActiveCodeSet(_ context.Context, _ string) (map[string]struct{}, error) {
	return map[string]struct{}{}, nil
}
func (s *stubOwnerRepo) GetStatusCodeSet(_ context.Context, _ string) (map[string]bool, error) {
	return map[string]bool{}, nil
}
func (s *stubOwnerRepo) Deactivate(_ context.Context, _, _, _ string) error { return nil }
func (s *stubOwnerRepo) Activate(_ context.Context, _, _ string) error      { return nil }
func (s *stubOwnerRepo) SoftDelete(_ context.Context, _, _, _ string) error { return nil }
func (s *stubOwnerRepo) GetOwnership(_ context.Context, _, _ string) (*postgres.URLOwnershipRecord, error) {
	return nil, nil
}

type stubShlinkClient struct {
	createResult *shlink.ShortURL
	createError  error
	updateResult *shlink.ShortURL
	updateError  error
	deleteError  error
	listResult   *shlink.ShortURLsResponse
	listError    error
}

func (s *stubShlinkClient) GetShortURLs(_ context.Context, _, _ string) (*shlink.ShortURLsResponse, error) {
	return s.listResult, s.listError
}
func (s *stubShlinkClient) GetShortURL(_ context.Context, _, _ string) (*shlink.ShortURL, error) {
	return s.createResult, s.createError
}
func (s *stubShlinkClient) GetShortURLVisits(_ context.Context, _, _, _, _ string, _ int) (*shlink.VisitsResponse, error) {
	return nil, nil
}
func (s *stubShlinkClient) CreateShortURL(_ context.Context, _ string, _ io.Reader) (*shlink.ShortURL, error) {
	return s.createResult, s.createError
}
func (s *stubShlinkClient) UpdateShortURL(_ context.Context, _, _ string, _ io.Reader) (*shlink.ShortURL, error) {
	return s.updateResult, s.updateError
}
func (s *stubShlinkClient) DeleteShortURL(_ context.Context, _, _ string) error {
	return s.deleteError
}
func (s *stubShlinkClient) GetTags(_ context.Context, _ string) (*shlink.TagsWithStatsResponse, error) {
	return nil, nil
}
func (s *stubShlinkClient) CreateTag(_ context.Context, _ string, _ io.Reader) (*shlink.TagsWithStatsResponse, error) {
	return nil, nil
}
func (s *stubShlinkClient) RenameTag(_ context.Context, _ string, _ io.Reader) error { return nil }
func (s *stubShlinkClient) DeleteTags(_ context.Context, _ string, _ []string) error  { return nil }
func (s *stubShlinkClient) GetNonOrphanVisits(_ context.Context, _, _, _ string, _ int) (*shlink.VisitsResponse, error) {
	return nil, nil
}
func (s *stubShlinkClient) PatchSettings(_ context.Context, _ string, _ int) error { return nil }
func (s *stubShlinkClient) GetHealth(_ context.Context) (*shlink.HealthResponse, error) {
	return &shlink.HealthResponse{Status: "pass", Version: "3.0.0"}, nil
}
func (s *stubShlinkClient) ValidateVersion(_ context.Context, _ int, _ int, _ time.Duration) error {
	return nil
}

// ── Helpers ────────────────────────────────────────────────────────────────────────────────────

func newTestShlinkSvc(client service.ShlinkClientIface) *service.ShlinkService {
	cfg := &config.Config{
		AdminRole:           "admin",
		ShlinkDefaultDomain: "http://localhost",
	}
	perms := service.NewStaticPermissionsCache(map[string]domain.RolePermissions{
		"user": {
			CanViewOwnLinks:              true,
			CanCreateLinks:               true,
			CanEditOwnLinks:              true,
			CanDeleteOwnLinks:            true,
			CanCreateWithoutSlug:         true,
			CanDeactivateOwnLinks:        true,
			CanReactivateOwnLinks:        true,
			CanDeleteOwnLinksPermanently: true,
		},
		"admin": {
			CanViewAllLinks:              true,
			CanEditAllLinks:              true,
			CanDeleteAllLinks:            true,
			CanDeactivateAllLinks:        true,
			CanReactivateAllLinks:        true,
			CanDeleteAllLinksPermanently: true,
		},
	})
	return service.NewShlinkService(client, cfg, perms)
}

func userCtx(r *http.Request, role string) *http.Request {
	u := &domain.User{
		Sub:          "sub-123",
		Username:     "testuser",
		Role:         domain.Role(role),
		ShlinkAPIKey: "test-api-key",
	}
	return r.WithContext(middleware.WithUser(r.Context(), u))
}

// ── Tests ───────────────────────────────────────────────────────────────────────────────────────

func TestCreateShortURL_success(t *testing.T) {
	client := &stubShlinkClient{
		createResult: &shlink.ShortURL{ShortCode: "abc", ShortURL: "http://s.test/abc"},
	}
	svc := newTestShlinkSvc(client)
	ownerRepo := &stubOwnerRepo{isOwner: true}
	cfg := &config.Config{ShlinkDefaultDomain: "http://localhost"}
	h := handler.NewShlinkProxyHandler(svc, nil, ownerRepo, cfg)

	body := bytes.NewBufferString(`{"longUrl":"https://example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/shlink/short-urls", body)
	req = userCtx(req, "user")
	rec := httptest.NewRecorder()

	h.CreateShortURL(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp shlink.ShortURL
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ShortCode != "abc" {
		t.Errorf("expected shortCode abc, got %s", resp.ShortCode)
	}
}

func TestDeleteShortURL_ownerSuccess(t *testing.T) {
	client := &stubShlinkClient{deleteError: nil}
	svc := newTestShlinkSvc(client)
	ownerRepo := &stubOwnerRepo{isOwner: true}
	cfg := &config.Config{ShlinkDefaultDomain: "http://localhost"}
	h := handler.NewShlinkProxyHandler(svc, nil, ownerRepo, cfg)

	r := chi.NewRouter()
	r.Delete("/api/shlink/short-urls/{shortCode}", h.DeleteShortURL)

	req := httptest.NewRequest(http.MethodDelete, "/api/shlink/short-urls/abc", nil)
	req = userCtx(req, "user")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteShortURL_notOwner(t *testing.T) {
	client := &stubShlinkClient{}
	svc := newTestShlinkSvc(client)
	ownerRepo := &stubOwnerRepo{isOwner: false}
	cfg := &config.Config{ShlinkDefaultDomain: "http://localhost"}
	h := handler.NewShlinkProxyHandler(svc, nil, ownerRepo, cfg)

	r := chi.NewRouter()
	r.Delete("/api/shlink/short-urls/{shortCode}", h.DeleteShortURL)

	req := httptest.NewRequest(http.MethodDelete, "/api/shlink/short-urls/abc", nil)
	req = userCtx(req, "user")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestListShortURLs_filtersByOwnership(t *testing.T) {
	client := &stubShlinkClient{
		listResult: &shlink.ShortURLsResponse{},
	}
	client.listResult.ShortURLs.Data = []shlink.ShortURL{
		{ShortCode: "abc"}, {ShortCode: "xyz"},
	}
	svc := newTestShlinkSvc(client)
	ownerRepo := &stubOwnerRepo{isOwner: true}
	cfg := &config.Config{ShlinkDefaultDomain: "http://localhost"}
	h := handler.NewShlinkProxyHandler(svc, nil, ownerRepo, cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/shlink/short-urls", nil)
	req = userCtx(req, "user")
	rec := httptest.NewRecorder()

	h.ListShortURLs(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}
