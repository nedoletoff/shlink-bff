package handler_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"unified-backend/internal/config"
	"unified-backend/internal/handler"
	"unified-backend/internal/repository/postgres"
	"unified-backend/internal/service"
	"unified-backend/internal/shlink"
)

// ── stubs ───────────────────────────────────────────────────────────────────────────────────

type stubOwnerRepo struct {
	isOwner    bool
	ownerError error
}

func (s *stubOwnerRepo) Save(_ interface{}, _, _, _, _ string) error            { return nil }
func (s *stubOwnerRepo) IsOwner(_ interface{}, _, _, _ string) (bool, error)   { return s.isOwner, s.ownerError }
func (s *stubOwnerRepo) HardDelete(_ interface{}, _, _ string) error           { return nil }
func (s *stubOwnerRepo) SetActive(_ interface{}, _, _ string, _ bool) error    { return nil }
func (s *stubOwnerRepo) GetShortCodeSet(_ interface{}, _ string) (map[string]struct{}, error) {
	return map[string]struct{}{}, nil
}
func (s *stubOwnerRepo) GetActiveCodeSet(_ interface{}, _ string) (map[string]struct{}, error) {
	return map[string]struct{}{}, nil
}
func (s *stubOwnerRepo) GetStatusCodeSet(_ interface{}, _ string) (map[string]bool, error) {
	return map[string]bool{}, nil
}
func (s *stubOwnerRepo) Deactivate(_ interface{}, _, _, _ string) error { return nil }
func (s *stubOwnerRepo) Activate(_ interface{}, _, _ string) error      { return nil }
func (s *stubOwnerRepo) SoftDelete(_ interface{}, _, _, _ string) error { return nil }
func (s *stubOwnerRepo) GetOwnership(_ interface{}, _, _ string) (*postgres.URLOwnershipRecord, error) {
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

func (s *stubShlinkClient) GetShortURLs(_ interface{}, _, _ string) (*shlink.ShortURLsResponse, error) {
	return s.listResult, s.listError
}
func (s *stubShlinkClient) GetShortURL(_ interface{}, _, _ string) (*shlink.ShortURL, error) {
	return s.createResult, s.createError
}
func (s *stubShlinkClient) GetShortURLVisits(_ interface{}, _, _, _, _ string, _ int) (*shlink.VisitsResponse, error) {
	return nil, nil
}
func (s *stubShlinkClient) CreateShortURL(_ interface{}, _ string, _ io.Reader) (*shlink.ShortURL, error) {
	return s.createResult, s.createError
}
func (s *stubShlinkClient) UpdateShortURL(_ interface{}, _, _ string, _ io.Reader) (*shlink.ShortURL, error) {
	return s.updateResult, s.updateError
}
func (s *stubShlinkClient) DeleteShortURL(_ interface{}, _, _ string) error { return s.deleteError }
func (s *stubShlinkClient) GetTags(_ interface{}, _ string) (*shlink.TagsWithStatsResponse, error) {
	return nil, nil
}
func (s *stubShlinkClient) CreateTag(_ interface{}, _ string, _ io.Reader) (*shlink.TagsWithStatsResponse, error) {
	return nil, nil
}
func (s *stubShlinkClient) RenameTag(_ interface{}, _ string, _ io.Reader) error { return nil }
func (s *stubShlinkClient) DeleteTags(_ interface{}, _ string, _ []string) error  { return nil }
func (s *stubShlinkClient) GetNonOrphanVisits(_ interface{}, _, _, _ string, _ int) (*shlink.VisitsResponse, error) {
	return nil, nil
}
func (s *stubShlinkClient) PatchSettings(_ interface{}, _ string, _ int) error { return nil }
func (s *stubShlinkClient) GetHealth(_ interface{}) (*shlink.HealthResponse, error) {
	return &shlink.HealthResponse{Status: "pass", Version: "3.0.0"}, nil
}
func (s *stubShlinkClient) ValidateVersion(_ interface{}, _ int, _ int, _ time.Duration) error {
	return nil
}

// permCtrl stub — реализует handler.PermChecker
type allowAllPerm struct{}

func (allowAllPerm) Check(_ interface{}, _ string, _ string) (bool, error) { return true, nil }

type denyAllPerm struct{}

func (denyAllPerm) Check(_ interface{}, _ string, _ string) (bool, error) { return false, nil }

// ── Хелпер ───────────────────────────────────────────────────────────────────────────────────

func newTestShlinkSvc(client service.ShlinkClientIface) *service.ShlinkService {
	cfg := &config.Config{
		ShlinkDefaultDomain: "http://localhost",
	}
	return service.NewShlinkService(client, cfg)
}

func newProxyHandler(svc *service.ShlinkService, ownerRepo handler.URLOwnershipIface, perm handler.PermChecker) *handler.ShlinkProxyHandler {
	cfg := &config.Config{ShlinkDefaultDomain: "http://localhost"}
	return handler.NewShlinkProxyHandler(svc, nil, ownerRepo, cfg, perm)
}

// ── Тесты ─────────────────────────────────────────────────────────────────────────────────────

func TestCreateShortURL_Success(t *testing.T) {
	client := &stubShlinkClient{
		createResult: &shlink.ShortURL{ShortCode: "abc", ShortURL: "http://s.test/abc"},
	}
	svc := newTestShlinkSvc(client)
	h := newProxyHandler(svc, &stubOwnerRepo{isOwner: true}, allowAllPerm{})

	body := bytes.NewBufferString(`{"longUrl":"https://example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/shlink/short-urls", body)
	req = userCtx(req)
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

func TestCreateShortURL_Forbidden(t *testing.T) {
	client := &stubShlinkClient{}
	svc := newTestShlinkSvc(client)
	h := newProxyHandler(svc, &stubOwnerRepo{}, denyAllPerm{})

	body := bytes.NewBufferString(`{"longUrl":"https://example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/shlink/short-urls", body)
	req = userCtx(req)
	rec := httptest.NewRecorder()

	h.CreateShortURL(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestDeleteShortURL_OwnerSuccess(t *testing.T) {
	client := &stubShlinkClient{deleteError: nil}
	svc := newTestShlinkSvc(client)
	h := newProxyHandler(svc, &stubOwnerRepo{isOwner: true}, allowAllPerm{})

	r := chi.NewRouter()
	r.Delete("/api/shlink/short-urls/{shortCode}", h.DeleteShortURL)

	req := httptest.NewRequest(http.MethodDelete, "/api/shlink/short-urls/abc", nil)
	req = userCtx(req)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteShortURL_NotOwner(t *testing.T) {
	client := &stubShlinkClient{}
	svc := newTestShlinkSvc(client)
	h := newProxyHandler(svc, &stubOwnerRepo{isOwner: false}, allowAllPerm{})

	r := chi.NewRouter()
	r.Delete("/api/shlink/short-urls/{shortCode}", h.DeleteShortURL)

	req := httptest.NewRequest(http.MethodDelete, "/api/shlink/short-urls/abc", nil)
	req = userCtx(req)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestListShortURLs_OK(t *testing.T) {
	client := &stubShlinkClient{
		listResult: &shlink.ShortURLsResponse{},
	}
	client.listResult.ShortURLs.Data = []shlink.ShortURL{
		{ShortCode: "abc"}, {ShortCode: "xyz"},
	}
	svc := newTestShlinkSvc(client)
	h := newProxyHandler(svc, &stubOwnerRepo{isOwner: true}, allowAllPerm{})

	req := httptest.NewRequest(http.MethodGet, "/api/shlink/short-urls", nil)
	req = userCtx(req)
	rec := httptest.NewRecorder()

	h.ListShortURLs(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}
