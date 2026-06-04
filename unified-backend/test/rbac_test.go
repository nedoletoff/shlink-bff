package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"unified-backend/internal/domain"
	"unified-backend/internal/middleware"
)

// defaultRoleGroups — дефолтный маппинг группа→роль, как в проде без ROLE_GROUPS.
// Соответствует defaultRoleGroups константе в config: "shlink-admins=admin,admin=admin".
func defaultRoleGroups() map[string]string {
	return map[string]string{
		"shlink-admins": "admin",
		"admin":         "admin",
	}
}

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
	handler := middleware.ExtractIdentity(defaultRoleGroups())(stub)

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
	handler := middleware.ExtractIdentity(defaultRoleGroups())(stub)

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

// captureRole — вспомогательная функция: возвращает Role из Identity после прохождения ExtractIdentity.
func captureRole(groups string, roleGroups map[string]string) string {
	var id *middleware.Identity
	h := middleware.ExtractIdentity(roleGroups)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id = middleware.IdentityFromCtx(r.Context())
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Auth-Request-User", "sub-test")
	if groups != "" {
		req.Header.Set("X-Auth-Request-Groups", groups)
	}
	h.ServeHTTP(httptest.NewRecorder(), req)
	if id == nil {
		return ""
	}
	return id.Role
}

// --- Дефолтные группы ---

func TestExtractIdentity_AdminGroup_Default(t *testing.T) {
	if role := captureRole("shlink-admins,developers", defaultRoleGroups()); role != "admin" {
		t.Errorf("expected admin, got %q", role)
	}
}

func TestExtractIdentity_AdminGroup_LegacyAdmin(t *testing.T) {
	if role := captureRole("admin", defaultRoleGroups()); role != "admin" {
		t.Errorf("expected admin (legacy group), got %q", role)
	}
}

// При дефолтном маппинге группы, не перечисленные в ROLE_GROUPS, возвращают пустую роль.
// Это ожидаемое поведение: RequireActiveUser заполнит её из БД или defaultUserPermissions.
func TestExtractIdentity_UnknownGroup_EmptyRole(t *testing.T) {
	if role := captureRole("developers,readonly", defaultRoleGroups()); role != "" {
		t.Errorf("expected empty role for unmapped groups, got %q", role)
	}
}

func TestExtractIdentity_EmptyGroups_EmptyRole(t *testing.T) {
	if role := captureRole("", defaultRoleGroups()); role != "" {
		t.Errorf("expected empty role for empty groups, got %q", role)
	}
}

// --- Кастомный ROLE_GROUPS ---

func TestExtractIdentity_CustomRoleGroups_Admin(t *testing.T) {
	custom := map[string]string{
		"shadmin":    "admin",
		"superusers": "admin",
		"editors":    "editor",
	}

	tests := []struct {
		groups string
		want   string
	}{
		{"shadmin", "admin"},
		{"superusers,devs", "admin"},
		{"editors", "editor"},
		{"shlink-admins", ""},   // старое имя не в кастомном маппинге
		{"admin", ""},           // тоже не в маппинге
		{"developers", ""},
	}
	for _, tc := range tests {
		got := captureRole(tc.groups, custom)
		if got != tc.want {
			t.Errorf("groups=%q: expected %q, got %q", tc.groups, tc.want, got)
		}
	}
}

func TestExtractIdentity_CaseInsensitive(t *testing.T) {
	custom := map[string]string{"shlink-admins": "admin"}
	if role := captureRole("SHLINK-ADMINS", custom); role != "admin" {
		t.Errorf("expected admin (case-insensitive), got %q", role)
	}
}

// Первая совпавшая группа в списке определяет роль.
func TestExtractIdentity_FirstMatchWins(t *testing.T) {
	custom := map[string]string{
		"shlink-admins": "admin",
		"editors":       "editor",
	}
	// пользователь в обеих группах — роль определяется первой по порядку в заголовке
	role := captureRole("editors,shlink-admins", custom)
	if role != "editor" {
		t.Errorf("expected editor (first match), got %q", role)
	}
}

// --- Заполнение полей Identity ---

func TestExtractIdentity_FieldsPopulated(t *testing.T) {
	var id *middleware.Identity
	handler := middleware.ExtractIdentity(defaultRoleGroups())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	if id.KeycloakRole != "admin" {
		t.Errorf("KeycloakRole: got %q", id.KeycloakRole)
	}
	if len(id.Groups) != 1 || id.Groups[0] != "shlink-admins" {
		t.Errorf("Groups: got %v", id.Groups)
	}
}

// KeycloakRole всегда отражает Keycloak-группы, даже если Role будет переписана из БД.
func TestExtractIdentity_KeycloakRoleAlwaysSet(t *testing.T) {
	var id *middleware.Identity
	custom := map[string]string{"shlink-admins": "admin"}
	handler := middleware.ExtractIdentity(custom)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id = middleware.IdentityFromCtx(r.Context())
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Auth-Request-User", "sub-x")
	req.Header.Set("X-Auth-Request-Groups", "shlink-admins")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if id.KeycloakRole != "admin" {
		t.Errorf("KeycloakRole should always be from Keycloak groups, got %q", id.KeycloakRole)
	}
	// Role и KeycloakRole совпадают до вмешательства RequireActiveUser
	if id.Role != id.KeycloakRole {
		t.Errorf("Role %q != KeycloakRole %q before RequireActiveUser", id.Role, id.KeycloakRole)
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

// TestClientIP_PrefersXRealIP — X-Real-IP от nginx имеет приоритет
func TestClientIP_PrefersXRealIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	req.Header.Set("X-Real-IP", "203.0.113.7")
	if got := middleware.ClientIP(req); got != "203.0.113.7" {
		t.Errorf("expected X-Real-IP, got %q", got)
	}
}

// TestClientIP_XForwardedForFirst — берётся первый IP из X-Forwarded-For
func TestClientIP_XForwardedForFirst(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "198.51.100.5, 10.0.0.1")
	if got := middleware.ClientIP(req); got != "198.51.100.5" {
		t.Errorf("expected first XFF IP, got %q", got)
	}
}

// TestClientIP_FallbackRemoteAddr — без заголовков используется RemoteAddr
func TestClientIP_FallbackRemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.0.2.9:6543"
	if got := middleware.ClientIP(req); got != "192.0.2.9:6543" {
		t.Errorf("expected RemoteAddr fallback, got %q", got)
	}
}
