package service

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const permCacheTTL = 5 * time.Minute

// --- L1: пермишни по имени роли ------------------------------------------

type roleCacheEntry struct {
	perms    []string
	expireAt time.Time
}

func (e roleCacheEntry) valid() bool { return time.Now().Before(e.expireAt) }

// --- L2: имя роли по userID -----------------------------------------------

type userRoleCacheEntry struct {
	roleName string
	expireAt time.Time
}

func (e userRoleCacheEntry) valid() bool { return time.Now().Before(e.expireAt) }

// PermissionService — проверка разрешений через JOIN users → roles → role_permissions → permissions.
//
// Кэш двухуровневый:
//   - L1 (rolePermCache): пермишни роли, общий для всех пользователей с той же ролью. TTL 5 мин.
//   - L2 (userRoleCache): имя роли по userID. TTL 5 мин. Не подвержен запросу в БД если оба валидны.
//
// Fallback: если users.role_id IS NULL — опрашиваем roles.name по users.role (строковое поле).
type PermissionService struct {
	pool *pgxpool.Pool

	// L1
	rolePermMu    sync.RWMutex
	rolePermCache map[string]roleCacheEntry

	// L2
	userRoleMu    sync.RWMutex
	userRoleCache map[uuid.UUID]userRoleCacheEntry
}

func NewPermissionService(pool *pgxpool.Pool) *PermissionService {
	return &PermissionService{
		pool:          pool,
		rolePermCache: make(map[string]roleCacheEntry),
		userRoleCache: make(map[uuid.UUID]userRoleCacheEntry),
	}
}

// GetUserPermissions возвращает пермишни пользователя через двухуровневый кэш.
func (s *PermissionService) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]string, error) {
	// Шаг 1: определить имя роли (L2)
	roleName, err := s.resolveRoleName(ctx, userID)
	if err != nil {
		return nil, err
	}
	if roleName == "" {
		// Пользователь ещё не провизионирован полностью — возвращаем пустой список.
		return nil, nil
	}

	// Шаг 2: пермишни роли (L1)
	return s.resolveRolePerms(ctx, roleName)
}

// resolveRoleName — определяет имя роли пользователя (L2 + БД-fallback).
func (s *PermissionService) resolveRoleName(ctx context.Context, userID uuid.UUID) (string, error) {
	// L2 hit
	s.userRoleMu.RLock()
	entry, ok := s.userRoleCache[userID]
	s.userRoleMu.RUnlock()
	if ok && entry.valid() {
		return entry.roleName, nil
	}

	// Опрашиваем БД: сначала через role_id JOIN, если NULL — fallback по users.role.
	const q = `
		SELECT COALESCE(
			(SELECT r.name FROM roles r WHERE r.id = u.role_id),
			u.role
		)
		FROM users u
		WHERE u.id = $1`

	var roleName string
	if err := s.pool.QueryRow(ctx, q, userID).Scan(&roleName); err != nil {
		return "", err
	}

	s.userRoleMu.Lock()
	s.userRoleCache[userID] = userRoleCacheEntry{
		roleName: roleName,
		expireAt: time.Now().Add(permCacheTTL),
	}
	s.userRoleMu.Unlock()

	return roleName, nil
}

// resolveRolePerms — пермишни роли (L1 + БД-fallback).
func (s *PermissionService) resolveRolePerms(ctx context.Context, roleName string) ([]string, error) {
	// L1 hit
	s.rolePermMu.RLock()
	entry, ok := s.rolePermCache[roleName]
	s.rolePermMu.RUnlock()
	if ok && entry.valid() {
		return entry.perms, nil
	}

	const q = `
		SELECT p.name
		FROM permissions p
		JOIN role_permissions_v2 rp ON rp.permission_id = p.id
		JOIN roles r               ON r.id = rp.role_id
		WHERE r.name = $1`

	rows, err := s.pool.Query(ctx, q, roleName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		perms = append(perms, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	s.rolePermMu.Lock()
	s.rolePermCache[roleName] = roleCacheEntry{
		perms:    perms,
		expireAt: time.Now().Add(permCacheTTL),
	}
	s.rolePermMu.Unlock()

	return perms, nil
}

// UserHasPermission проверяет, есть ли у пользователя конкретное разрешение.
func (s *PermissionService) UserHasPermission(ctx context.Context, userID uuid.UUID, action string) (bool, error) {
	perms, err := s.GetUserPermissions(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, p := range perms {
		if p == action {
			return true, nil
		}
	}
	return false, nil
}

// InvalidateUser сбрасывает L2-запись пользователя.
// Вызывай при смене role_id пользователя (PATCH /users/:id/role).
func (s *PermissionService) InvalidateUser(userID uuid.UUID) {
	s.userRoleMu.Lock()
	delete(s.userRoleCache, userID)
	s.userRoleMu.Unlock()
}

// InvalidateRole сбрасывает L1-запись роли + все L2-записи для этой роли.
// Вызывай при изменении разрешений роли (PUT /roles/:id/permissions).
func (s *PermissionService) InvalidateRole(roleName string) {
	s.rolePermMu.Lock()
	delete(s.rolePermCache, roleName)
	s.rolePermMu.Unlock()

	// Сбрасываем L2 для всех пользователей с той же ролью, чтобы они перечитали пермишни.
	s.userRoleMu.Lock()
	for id, e := range s.userRoleCache {
		if e.roleName == roleName {
			delete(s.userRoleCache, id)
		}
	}
	s.userRoleMu.Unlock()
}
