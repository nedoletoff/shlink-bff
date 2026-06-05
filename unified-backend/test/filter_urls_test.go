package test

import (
	"testing"

	"unified-backend/internal/domain"
	"unified-backend/internal/shlink"
)

// ── FilterShortURLsByUser ──────────────────────────────────────────────────

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

func TestFilterShortURLsByUser_EmptyList(t *testing.T) {
	svc := newShlinkService(false)
	user := &domain.User{Role: domain.RoleUser, Sub: "user1"}
	got := svc.FilterShortURLsByUser([]shlink.ShortURL{}, user)
	if len(got) != 0 {
		t.Errorf("empty input: want 0, got %d", len(got))
	}
}

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

func TestCanModifyShortCode_AdminCanModifyAll(t *testing.T) {
	svc := newShlinkService(false)
	admin := &domain.User{Role: domain.RoleAdmin, Sub: "admin1", SlugPrefix: "adm-"}

	for _, code := range []string{"adm-link", "user-link", "any-code", ""} {
		if !svc.CanModifyShortCode(admin, code, false) {
			t.Errorf("admin should be able to modify %q", code)
		}
	}
}

func TestCanModifyShortCode_UserOwnPrefix(t *testing.T) {
	svc := newShlinkService(true)
	user := &domain.User{Role: domain.RoleUser, Sub: "u1", SlugPrefix: "u1-"}

	if !svc.CanModifyShortCode(user, "u1-mylink", false) {
		t.Error("user should be able to modify own prefixed link")
	}
}

func TestCanModifyShortCode_UserForeignPrefix(t *testing.T) {
	svc := newShlinkService(true)
	user := &domain.User{Role: domain.RoleUser, Sub: "u1", SlugPrefix: "u1-"}

	if svc.CanModifyShortCode(user, "u2-otherlink", false) {
		t.Error("user should NOT be able to modify another user's link")
	}
}

func TestCanModifyShortCode_PrefixDisabled_UserCanModify(t *testing.T) {
	svc := newShlinkService(false)
	user := &domain.User{Role: domain.RoleUser, Sub: "u1", SlugPrefix: "u1-"}

	if !svc.CanModifyShortCode(user, "totally-foreign-code", false) {
		t.Error("when prefix disabled, user should be able to modify any code")
	}
}

func TestCanModifyShortCode_DeleteRequiresDeletePerm(t *testing.T) {
	svc := newShlinkServiceFull(false, false)
	user := &domain.User{Role: domain.RoleUser, Sub: "u1", SlugPrefix: "u1-"}

	// без CanDeleteOwnLinks — нельзя удалять даже свои
	if svc.CanModifyShortCode(user, "u1-mylink", true) {
		t.Error("user without delete perm should not be able to delete")
	}
}
