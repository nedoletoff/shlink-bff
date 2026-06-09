package service

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const permCacheTTL = 5 * time.Minute

type permCacheEntry struct {
	perms     []string
	cachedAt  time.Time
}

// PermissionService — проверка разрешений пользователя через JOIN users → roles → role_permissions → permissions.
// Результат кешируется на TTL=5мин и инвалидируется вручную через InvalidateUser.
type PermissionService struct {
	pool  *pgxpool.Pool
	mu    sync.RWMutex
	cache map[uuid.UUID]permCacheEntry
}

func NewPermissionService(pool *pgxpool.Pool) *PermissionService {
	return &PermissionService{
		pool:  pool,
		cache: make(map[uuid.UUID]permCacheEntry),
	}
}

// GetUserPermissions возвращает список имён разрешений для пользователя.
func (s *PermissionService) GetUserPermissions(ctx context.Context, userID uuid.UUID) ([]string, error) {
	s.mu.RLock()
	entry, ok := s.cache[userID]
	s.mu.RUnlock()

	if ok && time.Since(entry.cachedAt) < permCacheTTL {
		return entry.perms, nil
	}

	const q = `
		SELECT p.name
		FROM users u
		JOIN roles r              ON r.id = u.role_id
		JOIN role_permissions rp  ON rp.role_id = r.id
		JOIN permissions p        ON p.id = rp.permission_id
		WHERE u.id = $1`

	rows, err := s.pool.Query(ctx, q, userID)
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

	s.mu.Lock()
	s.cache[userID] = permCacheEntry{perms: perms, cachedAt: time.Now()}
	s.mu.Unlock()

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

// InvalidateUser сбрасывает кэш пользователя (call after role change).
func (s *PermissionService) InvalidateUser(userID uuid.UUID) {
	s.mu.Lock()
	delete(s.cache, userID)
	s.mu.Unlock()
}
