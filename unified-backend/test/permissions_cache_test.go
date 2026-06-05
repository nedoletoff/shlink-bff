package test

import (
	"context"
	"testing"

	"unified-backend/internal/domain"
	"unified-backend/internal/service"
)

// stubRolePermissionsRepo реализует минимальный stub для postgres.RolePermissionsRepository,
// нужный конструктору service.NewPermissionsCache.
// В unit-тестах БД не нужна: Load() вызывается напрямую с пустым или заполненным стабом.
type stubRolePermissionsRepo struct {
	data []domain.RolePermissions
	err  error
}

func (r *stubRolePermissionsRepo) GetAll(_ context.Context) ([]domain.RolePermissions, error) {
	return r.data, r.err
}

// newCache создаёт PermissionsCache с предзагруженными данными (без реальной БД).
func newCache(t *testing.T, rows []domain.RolePermissions, adminRole string) *service.PermissionsCache {
	t.Helper()
	cache := service.NewPermissionsCache(
		newStubRepo(rows),
		adminRole,
	)
	if err := cache.Load(context.Background()); err != nil {
		t.Fatalf("cache.Load: %v", err)
	}
	return cache
}

// --- Set + Get ---

// TestPermissionsCache_SetUpdatesCache проверяет, что Set инвалидирует кеш немедленно.
// Это ключевое требование: после PUT /api/admin/roles/{role}/permissions
// новые права должны применяться без рестарта.
func TestPermissionsCache_SetUpdatesCache(t *testing.T) {
	cache := newCache(t, nil, "admin")

	// до Set — роль отсутствует в кеше, получаем дефолт
	before := cache.Get("editor")
	if before.CanManageUsers {
		t.Fatal("editor should not have CanManageUsers before Set")
	}

	// устанавливаем кастомные права через Set (как это делает RolesHandler.UpsertRolePermissions)
	custom := domain.RolePermissions{
		Role:           "editor",
		CanViewOwnLinks: true,
		CanCreateLinks:  true,
		CanManageUsers:  true, // нетипично, но возможно
	}
	cache.Set(custom)

	after := cache.Get("editor")
	if after.Role != "editor" {
		t.Errorf("Role: got %q", after.Role)
	}
	if !after.CanCreateLinks {
		t.Error("CanCreateLinks should be true after Set")
	}
	if !after.CanManageUsers {
		t.Error("CanManageUsers should be true after Set")
	}
}

// TestPermissionsCache_SetOverwritesExisting — повторный Set перезаписывает старое значение.
func TestPermissionsCache_SetOverwritesExisting(t *testing.T) {
	initial := domain.RolePermissions{
		Role:           "editor",
		CanCreateLinks: true,
		CanEditOwnLinks: true,
	}
	cache := newCache(t, []domain.RolePermissions{initial}, "admin")

	// перезаписываем — убираем CanCreateLinks
	updated := domain.RolePermissions{
		Role:            "editor",
		CanCreateLinks:  false,
		CanEditOwnLinks: true,
	}
	cache.Set(updated)

	got := cache.Get("editor")
	if got.CanCreateLinks {
		t.Error("CanCreateLinks should be false after overwrite")
	}
	if !got.CanEditOwnLinks {
		t.Error("CanEditOwnLinks should still be true")
	}
}

// --- Get fallback ---

// TestPermissionsCache_Get_AdminFallback — adminRole не в БД → DefaultAdminPermissions.
func TestPermissionsCache_Get_AdminFallback(t *testing.T) {
	cache := newCache(t, nil, "admin") // пустая БД
	p := cache.Get("admin")

	if !p.CanManageUsers {
		t.Error("admin fallback: CanManageUsers should be true")
	}
	if !p.CanManageRoles {
		t.Error("admin fallback: CanManageRoles should be true")
	}
	if p.Role != "admin" {
		t.Errorf("admin fallback: Role should be admin, got %q", p.Role)
	}
}

// TestPermissionsCache_Get_UnknownRoleFallback — неизвестная роль → DefaultUserPermissions (deny elevated).
func TestPermissionsCache_Get_UnknownRoleFallback(t *testing.T) {
	cache := newCache(t, nil, "admin")
	p := cache.Get("stranger")

	if p.CanManageUsers {
		t.Error("unknown role fallback: CanManageUsers should be false")
	}
	if p.CanManageRoles {
		t.Error("unknown role fallback: CanManageRoles should be false")
	}
	// базовые права пользователя — должны быть
	if !p.CanViewOwnLinks {
		t.Error("unknown role fallback: CanViewOwnLinks should be true")
	}
}

// --- GetMerged (OR-семантика) ---

// TestPermissionsCache_GetMerged_OR — объединение прав нескольких ролей.
// Пользователь получает все флаги, разрешённые хотя бы в одной роли.
func TestPermissionsCache_GetMerged_OR(t *testing.T) {
	roles := []domain.RolePermissions{
		{
			Role:           "reader",
			CanViewOwnLinks: true,
			CanViewOwnStats: true,
		},
		{
			Role:           "creator",
			CanCreateLinks: true,
			CanEditOwnLinks: true,
		},
	}
	cache := newCache(t, roles, "admin")

	merged := cache.GetMerged([]string{"reader", "creator"})

	if !merged.CanViewOwnLinks {
		t.Error("merged: CanViewOwnLinks should be true (from reader)")
	}
	if !merged.CanCreateLinks {
		t.Error("merged: CanCreateLinks should be true (from creator)")
	}
	if !merged.CanEditOwnLinks {
		t.Error("merged: CanEditOwnLinks should be true (from creator)")
	}
	if !merged.CanViewOwnStats {
		t.Error("merged: CanViewOwnStats should be true (from reader)")
	}
	// флаги, которых нет ни в одной роли — false
	if merged.CanManageUsers {
		t.Error("merged: CanManageUsers should be false")
	}
	if merged.CanViewAllLinks {
		t.Error("merged: CanViewAllLinks should be false")
	}
}

// TestPermissionsCache_GetMerged_SingleRole — одна роль → те же права что Get.
func TestPermissionsCache_GetMerged_SingleRole(t *testing.T) {
	roles := []domain.RolePermissions{
		{Role: "editor", CanCreateLinks: true, CanEditOwnLinks: true},
	}
	cache := newCache(t, roles, "admin")

	merged := cache.GetMerged([]string{"editor"})
	direct := cache.Get("editor")

	if merged.CanCreateLinks != direct.CanCreateLinks {
		t.Error("GetMerged(single) should equal Get")
	}
	if merged.CanEditOwnLinks != direct.CanEditOwnLinks {
		t.Error("GetMerged(single) should equal Get")
	}
}

// TestPermissionsCache_GetMerged_Empty — пустой список ролей → нулевые права (deny-all).
func TestPermissionsCache_GetMerged_Empty(t *testing.T) {
	cache := newCache(t, nil, "admin")
	merged := cache.GetMerged([]string{})

	if merged.CanViewOwnLinks {
		t.Error("empty roles: CanViewOwnLinks should be false")
	}
	if merged.CanCreateLinks {
		t.Error("empty roles: CanCreateLinks should be false")
	}
}

// --- GetAll ---

// TestPermissionsCache_GetAll_ReturnsSnapshot — GetAll возвращает снимок всех ролей.
func TestPermissionsCache_GetAll_Empty(t *testing.T) {
	cache := newCache(t, nil, "admin")
	all := cache.GetAll()
	if len(all) != 0 {
		t.Errorf("expected 0 roles, got %d", len(all))
	}
}

func TestPermissionsCache_GetAll_AfterLoad(t *testing.T) {
	roles := []domain.RolePermissions{
		{Role: "editor", CanCreateLinks: true},
		{Role: "viewer", CanViewOwnLinks: true},
	}
	cache := newCache(t, roles, "admin")
	all := cache.GetAll()
	if len(all) != 2 {
		t.Errorf("expected 2 roles, got %d", len(all))
	}
}

func TestPermissionsCache_GetAll_AfterSet(t *testing.T) {
	cache := newCache(t, nil, "admin")
	cache.Set(domain.RolePermissions{Role: "editor", CanCreateLinks: true})
	cache.Set(domain.RolePermissions{Role: "viewer", CanViewOwnLinks: true})

	all := cache.GetAll()
	if len(all) != 2 {
		t.Errorf("expected 2 roles after Set, got %d", len(all))
	}
}

// --- Вспомогательные функции ---

// newStubRepo возвращает repo-совместимый stub, который возвращает заданные данные.
// Используется для инициализации PermissionsCache без реальной БД.
func newStubRepo(rows []domain.RolePermissions) service.RolePermissionsReader {
	return &stubRepoImpl{data: rows}
}

type stubRepoImpl struct {
	data []domain.RolePermissions
}

func (r *stubRepoImpl) GetAll(_ context.Context) ([]domain.RolePermissions, error) {
	return r.data, nil
}
