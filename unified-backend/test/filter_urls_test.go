package test

import (
	"context"
	"testing"

	"unified-backend/internal/domain"
	"unified-backend/internal/shlink"
)

// ── FilterShortURLsByUser ──────────────────────────────────────────────────

// TestFilterShortURLsByUser_AdminSeesAll — admin с CanViewAllLinks видит все ссылки.
func TestFilterShortURLsByUser_AdminSeesAll(t *testing.T) {
	svc := newShlinkService(false)
	admin := &domain.User{Role: domain.RoleAdmin, Sub: "admin1"}

	urls := []shlink.ShortURL{
		{ShortCode: "abc", Tags: []string{"user:user1"}},
		{ShortCode: "def", Tags: []string{"user:user2"}},
		{ShortCode: "ghi", Tags: []string{}},
	}

	got := svc.FilterShortURLsByUser(urls, admin)
	if len(got) != 3 {
		t.Errorf("admin: want 3 urls, got %d", len(got))
	}
}

// TestFilterShortURLsByUser_UserSeesOnlyOwn — обычный пользователь видит только свои ссылки.
func TestFilterShortURLsByUser_UserSeesOnlyOwn(t *testing.T) {
	svc := newShlinkServiceFull(false, false)
	user := &domain.User{Role: domain.RoleUser, Sub: "user1"}

	urls := []shlink.ShortURL{
		{ShortCode: "abc", Tags: []string{"user:user1"}},
		{ShortCode: "def", Tags: []string{"user:user2"}},
		{ShortCode: "ghi", Tags: []string{"user:user1", "extra"}},
	}

	got := svc.FilterShortURLsByUser(urls, user)
	if len(got) != 2 {
		t.Errorf("regular user: want 2 own urls, got %d", len(got))
	}
	for _, u := range got {
		found := false
		for _, tag := range u.Tags {
			if tag == "user:user1" {
				found = true
			}
		}
		if !found {
			t.Errorf("url %s does not belong to user1", u.ShortCode)
		}
	}
}

// TestFilterShortURLsByUser_EmptyList — пустой список → пустой результат.
func TestFilterShortURLsByUser_EmptyList(t *testing.T) {
	svc := newShlinkService(false)
	user := &domain.User{Role: domain.RoleUser, Sub: "user1"}
	got := svc.FilterShortURLsByUser([]shlink.ShortURL{}, user)
	if len(got) != 0 {
		t.Errorf("empty input: want 0, got %d", len(got))
	}
}

// TestFilterShortURLsByUser_NoMatchingTag — нет ссылок с тегом пользователя.
func TestFilterShortURLsByUser_NoMatchingTag(t *testing.T) {
	svc := newShlinkServiceFull(false, false)
	user := &domain.User{Role: domain.RoleUser, Sub: "user-orphan"}

	urls := []shlink.ShortURL{
		{ShortCode: "abc", Tags: []string{"user:someone-else"}},
		{ShortCode: "def", Tags: []string{}},
	}

	got := svc.FilterShortURLsByUser(urls, user)
	if len(got) != 0 {
		t.Errorf("no matching tag: want 0, got %d", len(got))
	}
}

// ── CanModifyShortCode ────────────────────────────────────────────────────

// TestCanModifyShortCode_AdminCanModifyAll — admin может изменить любой шорткод.
func TestCanModifyShortCode_AdminCanModifyAll(t *testing.T) {
	svc := newShlinkService(false)
	admin := &domain.User{Role: domain.RoleAdmin, Sub: "admin1", SlugPrefix: "adm-"}

	for _, code := range []string{"adm-link", "user-link", "any-code", ""} {
		if !svc.CanModifyShortCode(context.Background(), admin, code) {
			t.Errorf("admin should be able to modify %q", code)
		}
	}
}

// TestCanModifyShortCode_UserOwnPrefix — пользователь может изменять свои ссылки (совпадает prefix).
func TestCanModifyShortCode_UserOwnPrefix(t *testing.T) {
	svc := newShlinkService(true)
	user := &domain.User{Role: domain.RoleUser, Sub: "u1", SlugPrefix: "u1-"}

	if !svc.CanModifyShortCode(context.Background(), user, "u1-mylink") {
		t.Error("user should be able to modify own prefixed link")
	}
}

// TestCanModifyShortCode_UserForeignPrefix — пользователь не может изменить чужую ссылку.
func TestCanModifyShortCode_UserForeignPrefix(t *testing.T) {
	svc := newShlinkService(true)
	user := &domain.User{Role: domain.RoleUser, Sub: "u1", SlugPrefix: "u1-"}

	if svc.CanModifyShortCode(context.Background(), user, "u2-otherlink") {
		t.Error("user should NOT be able to modify another user's link")
	}
}

// TestCanModifyShortCode_PrefixDisabled_UserCanModify — при выключенном prefix
// пользователь может изменять любой код (контроль отсутствует).
func TestCanModifyShortCode_PrefixDisabled_UserCanModify(t *testing.T) {
	svc := newShlinkService(false)
	user := &domain.User{Role: domain.RoleUser, Sub: "u1", SlugPrefix: "u1-"}

	// без prefix enforcement — любой код разрешён
	if !svc.CanModifyShortCode(context.Background(), user, "totally-foreign-code") {
		t.Error("when prefix disabled, user should be able to modify any code")
	}
}
