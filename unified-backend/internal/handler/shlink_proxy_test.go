package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"unified-backend/internal/config"
	"unified-backend/internal/domain"
	"unified-backend/internal/handler"
	"unified-backend/internal/middleware"
	"unified-backend/internal/service"
	"unified-backend/internal/shlink"
)

// ─── stubs ────────────────────────────────────────────────────────────────────

type stubRolesRepoH struct{ data []domain.RolePermissions }

func (s *stubRolesRepoH) GetAll(_ context.Context) ([]domain.RolePermissions, error) {
	return s.data, nil
}

// stubOwnerRepo реализует интерфейс URLOwnershipRepository для тестов без БД.
// Использует простую in-memory map.
type stubOwnerRepo struct {
	// ownerSub → []shortCode
	byOwner  map[string][]string
	// shortCode → ownerSub
	ownedBy  map[string]string
	// softDeleted short_codes
	deleted  map[string]bool
	saveCalls int
	softDeleteCalls int
}

func newStubOwnerRepo() *stubOwnerRepo {
	return &stubOwnerRepo{
		byOwner: make(map[string][]string),
		ownedBy:  make(map[string]string),
		deleted:  make(map[string]bool),
	}
}

func (r *stubOwnerRepo) Save(_ context.Context, shortCode, ownerSub, _ string) error {
	r.saveCalls++
	r.ownedBy[shortCode] = ownerSub
	r.byOwner[ownerSub] = append(r.byOwner[ownerSub], shortCode)
	return nil
}

func (r *stubOwnerRepo) IsOwner(_ context.Context, shortCode, _ string, sub string) (bool, error) {
	if r.deleted[shortCode] {
		return false, nil
	}
	return r.ownedBy[shortCode] == sub, nil
}

func (r *stubOwnerRepo) SoftDelete(_ context.Context, shortCode, _, _ string) error {
	r.softDeleteCalls++
	r.deleted[shortCode] = true
	return nil
}

func (r *stubOwnerRepo) GetShortCodeSet(_ context.Context, ownerSub string) (map[string]struct{}, error) {
	set := make(map[string]struct{})
	for _, sc := range r.byOwner[ownerSub] {
		if !r.deleted[sc] {
			set[sc] = struct{}{}
		}
	}
	return set, nil
}

// stubShlinkServer запускает httptest.Server, симулирующий shlink API.
type stubShlinkServer struct {
	server       *httptest.Server
	createdCode  string
	updateCalled bool
	listResponse shlink.ShortURLsResponse
}

func newStubShlinkServer(code string, listURLs []shlink.ShortURL) *stubShlinkServer {
	s := &stubShlinkServer{createdCode: code}
	s.listResponse.ShortURLs.Data = listURLs

	mux := http.NewServeMux()
	mux.HandleFunc("/rest/v3/short-urls", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(s.listResponse)
		case http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(shlink.ShortURL{ShortCode: s.createdCode})
		}
	})
	mux.HandleFunc("/rest/v3/short-urls/", func(w http.ResponseWriter, r *http.Request) {
		s.updateCalled = true
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(shlink.ShortURL{ShortCode: s.createdCode})
	})
	s.server = httptest.NewServer(mux)
	return s
}

func (s *stubShlinkServer) Close() { s.server.Close() }

// ─── helpers ──────────────────────────────────────────────────────────────────

func newProxyHandler(
	perms []domain.RolePermissions,
	ownerRepo *stubOwnerRepo,
	shlinkBase string,
	cfg *config.Config,
) *handler.ShlinkProxyHandler {
	cache := service.NewPermissionsCache(&stubRolesRepoH{data: perms}, cfg.AdminRole)
	_ = cache.Load(context.Background())
	client := shlink.NewClient(shlinkBase)
	svc := service.NewShlinkService(client, cfg, cache)
	return handler.NewShlinkProxyHandler(svc, nil, ownerRepo, cfg)
}

func userReq(method, target string, user *domain.User, body []byte) *http.Request {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, target, bytes.NewReader(body))
	} else {
		r = httptest.NewRequest(method, target, nil)
	}
	ctx := middleware.WithUser(r.Context(), user)
	return r.WithContext(ctx)
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

// ─── ListShortURLs ─────────────────────────────────────────────────────────────

func TestListShortURLs_AdminSeesAll(t *testing.T) {
	listURLs := []shlink.ShortURL{
		{ShortCode: "aaa"},
		{ShortCode: "bbb"},
	}
	shlinkSrv := newStubShlinkServer("aaa", listURLs)
	defer shlinkSrv.Close()

	ownerRepo := newStubOwnerRepo()
	cfg := &config.Config{AdminRole: "admin", ShlinkURL: shlinkSrv.server.URL, ShlinkDefaultDomain: "http://s.local"}
	perms := []domain.RolePermissions{{Role: "admin", CanViewAllLinks: true, CanViewOwnLinks: true}}
	h := newProxyHandler(perms, ownerRepo, shlinkSrv.server.URL, cfg)

	user := &domain.User{Sub: "adm", Role: "admin", ShlinkAPIKey: "key"}
	rec := httptest.NewRecorder()
	req := userReq(http.MethodGet, "/api/shlink/short-urls", user, nil)
	h.ListShortURLs(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp shlink.ShortURLsResponse
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if len(resp.ShortURLs.Data) != 2 {
		t.Errorf("admin should see all 2, got %d", len(resp.ShortURLs.Data))
	}
}

func TestListShortURLs_UserSeesOnlyOwned(t *testing.T) {
	listURLs := []shlink.ShortURL{
		{ShortCode: "aaa"},
		{ShortCode: "bbb"},
	}
	shlinkSrv := newStubShlinkServer("aaa", listURLs)
	defer shlinkSrv.Close()

	ownerRepo := newStubOwnerRepo()
	_ = ownerRepo.Save(context.Background(), "aaa", "u1", "")

	cfg := &config.Config{AdminRole: "admin", ShlinkURL: shlinkSrv.server.URL, ShlinkDefaultDomain: "http://s.local"}
	perms := []domain.RolePermissions{{Role: "user", CanViewOwnLinks: true, CanViewAllLinks: false}}
	h := newProxyHandler(perms, ownerRepo, shlinkSrv.server.URL, cfg)

	user := &domain.User{Sub: "u1", Role: "user", ShlinkAPIKey: "key"}
	rec := httptest.NewRecorder()
	req := userReq(http.MethodGet, "/api/shlink/short-urls", user, nil)
	h.ListShortURLs(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp shlink.ShortURLsResponse
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if len(resp.ShortURLs.Data) != 1 {
		t.Errorf("user should see 1 owned, got %d", len(resp.ShortURLs.Data))
	}
	if resp.ShortURLs.Data[0].ShortCode != "aaa" {
		t.Errorf("expected aaa, got %s", resp.ShortURLs.Data[0].ShortCode)
	}
}

func TestListShortURLs_NoUser_Forbidden(t *testing.T) {
	shlinkSrv := newStubShlinkServer("", nil)
	defer shlinkSrv.Close()

	ownerRepo := newStubOwnerRepo()
	cfg := &config.Config{AdminRole: "admin", ShlinkURL: shlinkSrv.server.URL}
	h := newProxyHandler(nil, ownerRepo, shlinkSrv.server.URL, cfg)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/shlink/short-urls", nil)
	h.ListShortURLs(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

// ─── CreateShortURL ────────────────────────────────────────────────────────────

func TestCreateShortURL_SavesOwnership(t *testing.T) {
	shlinkSrv := newStubShlinkServer("newcode", nil)
	defer shlinkSrv.Close()

	ownerRepo := newStubOwnerRepo()
	cfg := &config.Config{AdminRole: "admin", ShlinkURL: shlinkSrv.server.URL, ShlinkDefaultDomain: "http://s.local"}
	perms := []domain.RolePermissions{
		{Role: "user", CanCreateLinks: true, CanCreateWithoutSlug: true, CanCreateWithCustomSlug: true},
	}
	h := newProxyHandler(perms, ownerRepo, shlinkSrv.server.URL, cfg)

	body, _ := json.Marshal(map[string]string{"longUrl": "https://example.com"})
	user := &domain.User{Sub: "u1", Role: "user", ShlinkAPIKey: "key"}
	rec := httptest.NewRecorder()
	req := userReq(http.MethodPost, "/api/shlink/short-urls", user, body)
	h.CreateShortURL(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if ownerRepo.saveCalls != 1 {
		t.Errorf("expected 1 save call, got %d", ownerRepo.saveCalls)
	}
	if ownerRepo.ownedBy["newcode"] != "u1" {
		t.Errorf("newcode should be owned by u1, got %q", ownerRepo.ownedBy["newcode"])
	}
}

func TestCreateShortURL_NoCreatePerm_Forbidden(t *testing.T) {
	shlinkSrv := newStubShlinkServer("x", nil)
	defer shlinkSrv.Close()

	ownerRepo := newStubOwnerRepo()
	cfg := &config.Config{AdminRole: "admin", ShlinkURL: shlinkSrv.server.URL}
	perms := []domain.RolePermissions{
		{Role: "readonly", CanCreateLinks: false},
	}
	h := newProxyHandler(perms, ownerRepo, shlinkSrv.server.URL, cfg)

	body, _ := json.Marshal(map[string]string{"longUrl": "https://example.com"})
	user := &domain.User{Sub: "u1", Role: "readonly", ShlinkAPIKey: "key"}
	rec := httptest.NewRecorder()
	req := userReq(http.MethodPost, "/api/shlink/short-urls", user, body)
	h.CreateShortURL(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

// ─── UpdateShortURL ────────────────────────────────────────────────────────────

func TestUpdateShortURL_OwnerCanEdit(t *testing.T) {
	shlinkSrv := newStubShlinkServer("abc", nil)
	defer shlinkSrv.Close()

	ownerRepo := newStubOwnerRepo()
	_ = ownerRepo.Save(context.Background(), "abc", "u1", "")

	cfg := &config.Config{AdminRole: "admin", ShlinkURL: shlinkSrv.server.URL, ShlinkDefaultDomain: "http://s.local"}
	perms := []domain.RolePermissions{
		{Role: "user", CanEditOwnLinks: true, CanEditAllLinks: false},
	}
	h := newProxyHandler(perms, ownerRepo, shlinkSrv.server.URL, cfg)

	body, _ := json.Marshal(map[string]string{"title": "new title"})
	user := &domain.User{Sub: "u1", Role: "user", ShlinkAPIKey: "key"}

	r := userReq(http.MethodPatch, "/api/shlink/short-urls/abc", user, body)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("shortCode", "abc")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	h.UpdateShortURL(rec, r)

	if rec.Code != http.StatusOK {
		t.Errorf("owner should be able to edit, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateShortURL_NonOwner_Forbidden(t *testing.T) {
	shlinkSrv := newStubShlinkServer("abc", nil)
	defer shlinkSrv.Close()

	ownerRepo := newStubOwnerRepo()
	_ = ownerRepo.Save(context.Background(), "abc", "u1", "") // owned by u1

	cfg := &config.Config{AdminRole: "admin", ShlinkURL: shlinkSrv.server.URL, ShlinkDefaultDomain: "http://s.local"}
	perms := []domain.RolePermissions{
		{Role: "user", CanEditOwnLinks: true, CanEditAllLinks: false},
	}
	h := newProxyHandler(perms, ownerRepo, shlinkSrv.server.URL, cfg)

	body, _ := json.Marshal(map[string]string{"title": "new title"})
	user := &domain.User{Sub: "u2", Role: "user", ShlinkAPIKey: "key"} // u2 tries to edit u1's link

	r := userReq(http.MethodPatch, "/api/shlink/short-urls/abc", user, body)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("shortCode", "abc")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	h.UpdateShortURL(rec, r)

	if rec.Code != http.StatusForbidden {
		t.Errorf("non-owner should be forbidden, got %d", rec.Code)
	}
}

// ─── DeleteShortURL ────────────────────────────────────────────────────────────

func TestDeleteShortURL_SoftDeletesOwnerRecord(t *testing.T) {
	shlinkSrv := newStubShlinkServer("del1", nil)
	defer shlinkSrv.Close()

	ownerRepo := newStubOwnerRepo()
	_ = ownerRepo.Save(context.Background(), "del1", "u1", "")

	cfg := &config.Config{AdminRole: "admin", ShlinkURL: shlinkSrv.server.URL, ShlinkDefaultDomain: "http://s.local"}
	perms := []domain.RolePermissions{
		{Role: "user", CanDeleteOwnLinks: true, CanDeleteAllLinks: false},
	}
	h := newProxyHandler(perms, ownerRepo, shlinkSrv.server.URL, cfg)

	user := &domain.User{Sub: "u1", Role: "user", ShlinkAPIKey: "key"}
	r := userReq(http.MethodDelete, "/api/shlink/short-urls/del1", user, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("shortCode", "del1")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	h.DeleteShortURL(rec, r)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	if ownerRepo.softDeleteCalls != 1 {
		t.Errorf("expected 1 soft delete call, got %d", ownerRepo.softDeleteCalls)
	}
	if !ownerRepo.deleted["del1"] {
		t.Error("del1 should be marked as deleted")
	}
}

func TestDeleteShortURL_NonOwner_Forbidden(t *testing.T) {
	shlinkSrv := newStubShlinkServer("del2", nil)
	defer shlinkSrv.Close()

	ownerRepo := newStubOwnerRepo()
	_ = ownerRepo.Save(context.Background(), "del2", "u1", "")

	cfg := &config.Config{AdminRole: "admin", ShlinkURL: shlinkSrv.server.URL, ShlinkDefaultDomain: "http://s.local"}
	perms := []domain.RolePermissions{
		{Role: "user", CanDeleteOwnLinks: true, CanDeleteAllLinks: false},
	}
	h := newProxyHandler(perms, ownerRepo, shlinkSrv.server.URL, cfg)

	user := &domain.User{Sub: "u2", Role: "user", ShlinkAPIKey: "key"}
	r := userReq(http.MethodDelete, "/api/shlink/short-urls/del2", user, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("shortCode", "del2")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	h.DeleteShortURL(rec, r)

	if rec.Code != http.StatusForbidden {
		t.Errorf("non-owner delete should be 403, got %d", rec.Code)
	}
	if ownerRepo.softDeleteCalls != 0 {
		t.Error("soft delete should not be called for non-owner")
	}
}

func TestDeleteShortURL_TombstonePatchCalled(t *testing.T) {
	shlinkSrv := newStubShlinkServer("tomb1", nil)
	defer shlinkSrv.Close()

	ownerRepo := newStubOwnerRepo()
	_ = ownerRepo.Save(context.Background(), "tomb1", "u1", "")

	cfg := &config.Config{AdminRole: "admin", ShlinkURL: shlinkSrv.server.URL, ShlinkDefaultDomain: "http://s.local"}
	perms := []domain.RolePermissions{
		{Role: "user", CanDeleteOwnLinks: true, CanDeleteAllLinks: false},
	}
	h := newProxyHandler(perms, ownerRepo, shlinkSrv.server.URL, cfg)

	user := &domain.User{Sub: "u1", Role: "user", ShlinkAPIKey: "key"}
	r := userReq(http.MethodDelete, "/api/shlink/short-urls/tomb1", user, nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("shortCode", "tomb1")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	h.DeleteShortURL(rec, r)

	if !shlinkSrv.updateCalled {
		t.Error("expected tombstone PATCH to be called on shlink")
	}
}
