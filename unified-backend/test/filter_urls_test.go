package test

import (
	"testing"

	"unified-backend/internal/domain"
	"unified-backend/internal/shlink"
)

// ── FilterShortURLsByUser ──────────────────────────────────────────────────

func TestFilterShortURLsByUser_AdminSeesAll(t *testing.T) {
	svc := newShlinkService(true)
	admin := &domain.User{Role: domain.RoleAdmin, SlugPrefix: "adm-"}

	urls := []shlink.ShortURL{
		{ShortCode: "adm-link"},
		{ShortCode: "u1-abc"},
		{ShortCode: "random"},
	}

	got := svc.FilterShortURLsByUser(urls, admin)
	if len(got) != 3 {
		t.Errorf("admin: want 3 urls, got %d", len(got))
	}
}

func TestFilterShortURLsByUser_UserSeesOnlyOwnPrefix(t *testing.T) {
	svc := newShlinkService(true)
	user := &domain.User{Role: domain.RoleUser, Sub: "u1", SlugPrefix: "u1-"}

	urls := []shlink.ShortURL{
		{ShortCode: "u1-abc"},
		{ShortCode: "u1-xyz"},
		{ShortCode: "u2-abc"},
		{ShortCode: "random"},
	}

	got := svc.FilterShortURLsByUser(urls, user)
	if len(got) != 2 {
		t.Errorf("regular user: want 2 own urls, got %d", len(got))
	}
	for _, u := range got {
		if len(u.ShortCode) < 3 || u.ShortCode[:3] != "u1-" {
			t.Errorf("url %q does not start with prefix u1-", u.ShortCode)
		}
	}
}

func TestFilterShortURLsByUser_EmptyList(t *testing.T) {
	svc := newShlinkService(true)
	user := &domain.User{Role: domain.RoleUser, Sub: "u1", SlugPrefix: "u1-"}
	got := svc.FilterShortURLsByUser([]shlink.ShortURL{}, user)
	if len(got) != 0 {
		t.Errorf("empty input: want 0, got %d", len(got))
	}
}

func TestFilterShortURLsByUser_NoMatchingPrefix(t *testing.T) {
	svc := newShlinkService(true)
	user := &domain.User{Role: domain.RoleUser, Sub: "orphan", SlugPrefix: "orphan-"}

	urls := []shlink.ShortURL{
		{ShortCode: "u1-abc"},
		{ShortCode: "u2-abc"},
	}

	got := svc.FilterShortURLsByUser(urls, user)
	if len(got) != 0 {
		t.Errorf("no matching prefix: want 0, got %d", len(got))
	}
}

func TestFilterShortURLsByUser_FeatureDisabled_UserSeesAll(t *testing.T) {
	// При slugPrefixEnabled=false фильтрация по prefix не применяется
	svc := newShlinkService(false)
	user := &domain.User{Role: domain.RoleUser, Sub: "u1", SlugPrefix: "u1-"}

	urls := []shlink.ShortURL{
		{ShortCode: "u1-abc"},
		{ShortCode: "u2-foreign"},
		{ShortCode: "random"},
	}

	got := svc.FilterShortURLsByUser(urls, user)
	if len(got) != 3 {
		t.Errorf("feature disabled: user should see all 3, got %d", len(got))
	}
}

// ── CanModifyShortCode ─────────────────────────────────────────────────────

func TestCanModifyShortCode_AdminCanModifyAll(t *testing.T) {
	svc := newShlinkService(true)
	admin := &domain.User{Role: domain.RoleAdmin, Sub: "admin1", SlugPrefix: "adm-"}

	for _, code := range []string{"adm-link", "u1-link", "any-code"} {
		if !svc.CanModifyShortCode(admin, code, false) {
			t.Errorf("admin should be able to edit %q", code)
		}
		if !svc.CanModifyShortCode(admin, code, true) {
			t.Errorf("admin should be able to delete %q", code)
		}
	}
}

func TestCanModifyShortCode_UserOwnPrefix(t *testing.T) {
	svc := newShlinkService(true)
	user := &domain.User{Role: domain.RoleUser, Sub: "u1", SlugPrefix: "u1-"}

	if !svc.CanModifyShortCode(user, "u1-mylink", false) {
		t.Error("user should be able to edit own prefixed link")
	}
	if !svc.CanModifyShortCode(user, "u1-mylink", true) {
		t.Error("user should be able to delete own prefixed link")
	}
}

func TestCanModifyShortCode_UserForeignPrefix(t *testing.T) {
	svc := newShlinkService(true)
	user := &domain.User{Role: domain.RoleUser, Sub: "u1", SlugPrefix: "u1-"}

	if svc.CanModifyShortCode(user, "u2-otherlink", false) {
		t.Error("user should NOT be able to edit another user's link")
	}
	if svc.CanModifyShortCode(user, "u2-otherlink", true) {
		t.Error("user should NOT be able to delete another user's link")
	}
}

func TestCanModifyShortCode_PrefixDisabled_UserCanModifyAny(t *testing.T) {
	svc := newShlinkService(false)
	user := &domain.User{Role: domain.RoleUser, Sub: "u1", SlugPrefix: "u1-"}

	if !svc.CanModifyShortCode(user, "totally-foreign-code", false) {
		t.Error("when prefix disabled, user should be able to edit any code")
	}
	if !svc.CanModifyShortCode(user, "totally-foreign-code", true) {
		t.Error("when prefix disabled, user should be able to delete any code")
	}
}

func TestCanModifyShortCode_UserNoPrefix_Denied(t *testing.T) {
	svc := newShlinkService(true)
	user := &domain.User{Role: domain.RoleUser, Sub: "u1", SlugPrefix: ""}

	if svc.CanModifyShortCode(user, "any-code", false) {
		t.Error("user with no prefix and feature enabled must be denied (edit)")
	}
	if svc.CanModifyShortCode(user, "any-code", true) {
		t.Error("user with no prefix and feature enabled must be denied (delete)")
	}
}
