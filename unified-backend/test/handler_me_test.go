package test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"unified-backend/internal/config"
	"unified-backend/internal/domain"
	"unified-backend/internal/handler"
	"unified-backend/internal/middleware"

	"github.com/google/uuid"
)

func buildUserContext(user *domain.User) context.Context {
	ctx := context.Background()
	ctx = middleware.WithUser(ctx, user)
	return ctx
}

// TestMeHandler_ReturnsCorrectFields — /api/me не возвращает shlink_api_key
func TestMeHandler_ReturnsCorrectFields(t *testing.T) {
	cfg := &config.Config{
		UserSlugPrefixEnabled:    true,
		UserTagInternalIdEnabled: false,
	}
	h := handler.NewMeHandler(cfg)

	user := &domain.User{
		ID:           uuid.New(),
		Sub:          "sub-test-123",
		Username:     "nikita",
		Email:        "nikita@example.com",
		Role:         domain.RoleUser,
		ShlinkAPIKey: "secret-api-key-DO-NOT-LEAK",
		SlugPrefix:   "n1-",
		Status:       domain.StatusActive,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req = req.WithContext(middleware.WithUser(req.Context(), user))
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// API key НИКОГДА не должен присутствовать в ответе
	if _, exists := resp["shlinkApiKey"]; exists {
		t.Error("SECURITY VIOLATION: shlinkApiKey must never appear in /api/me response")
	}
	if _, exists := resp["shlink_api_key"]; exists {
		t.Error("SECURITY VIOLATION: shlink_api_key must never appear in /api/me response")
	}
	if _, exists := resp["apiKey"]; exists {
		t.Error("SECURITY VIOLATION: apiKey must never appear in /api/me response")
	}

	// Обязательные поля
	if resp["sub"] != "sub-test-123" {
		t.Errorf("sub mismatch: %v", resp["sub"])
	}
	if resp["username"] != "nikita" {
		t.Errorf("username mismatch: %v", resp["username"])
	}
	if resp["role"] != "user" {
		t.Errorf("role mismatch: %v", resp["role"])
	}

	// hasApiKey должен быть true (ключ задан), но сам ключ не виден
	if resp["hasApiKey"] != true {
		t.Error("hasApiKey should be true when API key is set")
	}

	// features должны присутствовать
	features, ok := resp["features"].(map[string]any)
	if !ok {
		t.Fatal("features should be an object")
	}
	if features["userSlugPrefixEnabled"] != true {
		t.Error("userSlugPrefixEnabled should match config")
	}

	// permissions должны присутствовать
	perms, ok := resp["permissions"].(map[string]any)
	if !ok {
		t.Fatal("permissions should be an object")
	}
	if perms["canManageUsers"] != false {
		t.Error("user role should not canManageUsers")
	}
	if perms["canViewAuditLogs"] != false {
		t.Error("user role should not canViewAuditLogs")
	}
}

// TestMeHandler_AdminPermissions — admin получает расширенные права
func TestMeHandler_AdminPermissions(t *testing.T) {
	cfg := &config.Config{}
	h := handler.NewMeHandler(cfg)

	admin := &domain.User{
		ID:           uuid.New(),
		Sub:          "admin-sub",
		Username:     "admin",
		Email:        "admin@example.com",
		Role:         domain.RoleAdmin,
		ShlinkAPIKey: "admin-key-never-leak",
		Status:       domain.StatusActive,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req = req.WithContext(middleware.WithUser(req.Context(), admin))
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	var resp map[string]any
	_ = json.NewDecoder(rr.Body).Decode(&resp)

	perms := resp["permissions"].(map[string]any)
	if perms["canManageUsers"] != true {
		t.Error("admin should canManageUsers")
	}
	if perms["canViewAuditLogs"] != true {
		t.Error("admin should canViewAuditLogs")
	}

	// Проверяем отсутствие любого варианта API key
	for _, field := range []string{"shlinkApiKey", "shlink_api_key", "apiKey", "api_key"} {
		if _, exists := resp[field]; exists {
			t.Errorf("SECURITY VIOLATION: field %q must not be in response", field)
		}
	}
}

// TestMeHandler_NoUser_InternalError — без user в контексте → 500
func TestMeHandler_NoUser_InternalError(t *testing.T) {
	cfg := &config.Config{}
	h := handler.NewMeHandler(cfg)

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	// Намеренно НЕ кладём user в контекст
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 without user in context, got %d", rr.Code)
	}
}
