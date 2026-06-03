package test

import (
	"context"
	"testing"

	"unified-backend/internal/config"
	"unified-backend/internal/domain"
	"unified-backend/internal/service"
	"unified-backend/internal/shlink"
)

func newShlinkService(slugPrefixEnabled bool) *service.ShlinkService {
	cfg := &config.Config{
		UserSlugPrefixEnabled:    slugPrefixEnabled,
		UserTagInternalIdEnabled: false,
		ShlinkURL:                "http://shlink-api:8080",
	}
	cli := shlink.NewClient(cfg.ShlinkURL)
	return service.NewShlinkService(cli, cfg)
}

// ─── EnforceSlugPrefix ────────────────────────────────────────────────────────

// TestEnforceSlugPrefix_AdminBypass — для admin prefix не применяется
func TestEnforceSlugPrefix_AdminBypass(t *testing.T) {
	svc := newShlinkService(true)
	admin := &domain.User{Role: domain.RoleAdmin, SlugPrefix: "adm-"}
	slug := "my-custom-slug"

	result, err := svc.EnforceSlugPrefix(context.Background(), admin, &slug)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != slug {
		t.Errorf("admin slug should be unchanged: expected %q, got %q", slug, result)
	}
}

// TestEnforceSlugPrefix_CustomRoleBypass — кастомная роль без admin-прав тоже использует prefix
func TestEnforceSlugPrefix_CustomRoleBypass_IsNotAdmin(t *testing.T) {
	svc := newShlinkService(true)
	viewer := &domain.User{Role: domain.Role("viewer"), SlugPrefix: "v1-"}
	slug := "v1-mylink"

	result, err := svc.EnforceSlugPrefix(context.Background(), viewer, &slug)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "v1-mylink" {
		t.Errorf("expected %q, got %q", "v1-mylink", result)
	}
}

// TestEnforceSlugPrefix_UserNoPrefix — feature enabled, нет prefix → ошибка
func TestEnforceSlugPrefix_UserNoPrefix(t *testing.T) {
	svc := newShlinkService(true)
	user := &domain.User{Role: domain.Role("user"), SlugPrefix: ""}
	slug := "my-slug"

	_, err := svc.EnforceSlugPrefix(context.Background(), user, &slug)
	if err == nil {
		t.Error("expected error when user has no slug prefix")
	}
}

// TestEnforceSlugPrefix_CustomRoleNoPrefix — кастомная роль без prefix → ошибка
func TestEnforceSlugPrefix_CustomRoleNoPrefix(t *testing.T) {
	svc := newShlinkService(true)
	editor := &domain.User{Role: domain.Role("editor"), SlugPrefix: ""}
	slug := "my-slug"

	_, err := svc.EnforceSlugPrefix(context.Background(), editor, &slug)
	if err == nil {
		t.Error("expected error when editor has no slug prefix")
	}
}

// TestEnforceSlugPrefix_UserCorrectPrefix — slug с правильным prefix → OK
func TestEnforceSlugPrefix_UserCorrectPrefix(t *testing.T) {
	svc := newShlinkService(true)
	user := &domain.User{Role: domain.Role("user"), SlugPrefix: "u1-"}
	slug := "u1-mylink"

	result, err := svc.EnforceSlugPrefix(context.Background(), user, &slug)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "u1-mylink" {
		t.Errorf("expected %q, got %q", "u1-mylink", result)
	}
}

// TestEnforceSlugPrefix_UserWrongPrefix — slug без prefix → ошибка
func TestEnforceSlugPrefix_UserWrongPrefix(t *testing.T) {
	svc := newShlinkService(true)
	user := &domain.User{Role: domain.Role("user"), SlugPrefix: "u1-"}
	slug := "admin-link"

	_, err := svc.EnforceSlugPrefix(context.Background(), user, &slug)
	if err == nil {
		t.Error("expected error for slug without correct prefix")
	}
}

// TestEnforceSlugPrefix_FeatureDisabled — feature выключен → slug не трогается
func TestEnforceSlugPrefix_FeatureDisabled(t *testing.T) {
	svc := newShlinkService(false)
	user := &domain.User{Role: domain.Role("user"), SlugPrefix: "u1-"}
	slug := "any-slug"

	result, err := svc.EnforceSlugPrefix(context.Background(), user, &slug)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != slug {
		t.Errorf("when feature disabled, slug should be unchanged: got %q", result)
	}
}

// TestEnforceSlugPrefix_UserNilSlug — nil slug + prefix → возвращает prefix
func TestEnforceSlugPrefix_UserNilSlug(t *testing.T) {
	svc := newShlinkService(true)
	user := &domain.User{Role: domain.Role("user"), SlugPrefix: "u2-"}

	result, err := svc.EnforceSlugPrefix(context.Background(), user, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "u2-" {
		t.Errorf("expected prefix %q, got %q", "u2-", result)
	}
}

// TestEnforceSlugPrefix_CustomRole_NilSlug — кастомная роль, nil slug → prefix
func TestEnforceSlugPrefix_CustomRole_NilSlug(t *testing.T) {
	svc := newShlinkService(true)
	viewer := &domain.User{Role: domain.Role("viewer"), SlugPrefix: "vw-"}

	result, err := svc.EnforceSlugPrefix(context.Background(), viewer, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "vw-" {
		t.Errorf("expected prefix %q, got %q", "vw-", result)
	}
}

// ─── FilterShortURLsByUser ────────────────────────────────────────────────────

// TestFilterShortURLsByUser — фильтрация по prefix для роли user
func TestFilterShortURLsByUser(t *testing.T) {
	svc := newShlinkService(true)
	user := &domain.User{Role: domain.Role("user"), SlugPrefix: "u1-"}

	urls := []shlink.ShortURL{
		{ShortCode: "u1-abc"},
		{ShortCode: "u1-xyz"},
		{ShortCode: "u2-abc"},
		{ShortCode: "random"},
	}

	filtered := svc.FilterShortURLsByUser(urls, user)

	if len(filtered) != 2 {
		t.Errorf("expected 2 filtered URLs, got %d", len(filtered))
	}
	for _, u := range filtered {
		if len(u.ShortCode) < 3 || u.ShortCode[:3] != "u1-" {
			t.Errorf("expected prefix u1-, got %s", u.ShortCode)
		}
	}
}

// TestFilterShortURLsByUser_CustomRole — кастомная роль фильтрует по своему prefix
func TestFilterShortURLsByUser_CustomRole(t *testing.T) {
	svc := newShlinkService(true)
	viewer := &domain.User{Role: domain.Role("viewer"), SlugPrefix: "vw-"}

	urls := []shlink.ShortURL{
		{ShortCode: "vw-abc"},
		{ShortCode: "vw-xyz"},
		{ShortCode: "u1-abc"},
		{ShortCode: "random"},
	}

	result := svc.FilterShortURLsByUser(urls, viewer)
	if len(result) != 2 {
		t.Errorf("viewer should see 2 URLs, got %d", len(result))
	}
}

// TestFilterShortURLsByUser_AdminGetAll — admin видит все
func TestFilterShortURLsByUser_AdminGetAll(t *testing.T) {
	svc := newShlinkService(true)
	admin := &domain.User{Role: domain.RoleAdmin, SlugPrefix: ""}

	urls := []shlink.ShortURL{
		{ShortCode: "u1-abc"},
		{ShortCode: "u2-xyz"},
		{ShortCode: "random"},
	}

	result := svc.FilterShortURLsByUser(urls, admin)
	if len(result) != 3 {
		t.Errorf("admin should see all URLs, got %d", len(result))
	}
}

// TestFilterShortURLsByUser_EmptyPrefix — пустой prefix → возвращает все (не фильтрует)
func TestFilterShortURLsByUser_EmptyPrefix(t *testing.T) {
	svc := newShlinkService(true)
	user := &domain.User{Role: domain.Role("user"), SlugPrefix: ""}

	urls := []shlink.ShortURL{
		{ShortCode: "abc"},
		{ShortCode: "xyz"},
	}
	result := svc.FilterShortURLsByUser(urls, user)
	if len(result) != 2 {
		t.Errorf("empty prefix should return all, got %d", len(result))
	}
}

// TestFilterShortURLsByUser_FeatureDisabled — feature выключен → все видят всё
func TestFilterShortURLsByUser_FeatureDisabled(t *testing.T) {
	svc := newShlinkService(false)
	user := &domain.User{Role: domain.Role("user"), SlugPrefix: "u1-"}

	urls := []shlink.ShortURL{
		{ShortCode: "u1-abc"},
		{ShortCode: "u2-xyz"},
	}
	result := svc.FilterShortURLsByUser(urls, user)
	if len(result) != 2 {
		t.Errorf("feature disabled: should return all 2, got %d", len(result))
	}
}
