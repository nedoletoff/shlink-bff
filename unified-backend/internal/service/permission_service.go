package service

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultCacheTTL = 5 * time.Minute

// cacheEntry — запись кэша для одного пользователя
type cacheEntry struct {
	perms  []string
	expiresAt time.Time
}

// PermissionService работает с RBAC-таблицами permissions / roles / role_permissions_v2.
type PermissionService struct {
	db  *pgxpool.Pool
	ttl time.Duration

	mu    sync.RWMutex
	cache map[string]cacheEntry // key = userSub
}

type Option func(*PermissionService)

// WithCacheTTL переопределяет TTL кэша (по умолчанию 5 минут).
func WithCacheTTL(ttl time.Duration) Option {
	return func(s *PermissionService) { s.ttl = ttl }
}

func NewPermissionService(db *pgxpool.Pool, opts ...Option) *PermissionService {
	s := &PermissionService{
		db:    db,
		ttl:   defaultCacheTTL,
		cache: make(map[string]cacheEntry),
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// GetUserPermissions возвращает список имён разрешений пользователя (из кэша или БД).
func (s *PermissionService) GetUserPermissions(ctx context.Context, userSub string) ([]string, error) {
	if perms, ok := s.fromCache(userSub); ok {
		return perms, nil
	}
	return s.loadAndCache(ctx, userSub)
}

// UserHasPermission проверяет наличие разрешения у пользователя.
func (s *PermissionService) UserHasPermission(ctx context.Context, userSub, action string) (bool, error) {
	perms, err := s.GetUserPermissions(ctx, userSub)
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

// InvalidateUser сбрасывает кэш конкретного пользователя.
// Вызывается при PATCH /users/:sub/role.
func (s *PermissionService) InvalidateUser(userSub string) {
	s.mu.Lock()
	delete(s.cache, userSub)
	s.mu.Unlock()
}

// InvalidateAll очищает весь кэш (например, при изменении разрешений роли).
func (s *PermissionService) InvalidateAll() {
	s.mu.Lock()
	s.cache = make(map[string]cacheEntry)
	s.mu.Unlock()
}

// ──────────────────────────────────────────────────────────────────────────────
func (s *PermissionService) fromCache(sub string) ([]string, bool) {
	s.mu.RLock()
	e, ok := s.cache[sub]
	s.mu.RUnlock()
	if !ok || time.Now().After(e.expiresAt) {
		return nil, false
	}
	return e.perms, true
}

func (s *PermissionService) loadAndCache(ctx context.Context, sub string) ([]string, error) {
	rows, err := s.db.Query(ctx, `
		SELECT p.name
		FROM   users u
		JOIN   roles             r  ON r.id  = u.role_id
		JOIN   role_permissions_v2 rp ON rp.role_id = r.id
		JOIN   permissions       p  ON p.id  = rp.permission_id
		WHERE  u.sub = $1
	`, sub)
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
	if perms == nil {
		perms = []string{}
	}

	s.mu.Lock()
	s.cache[sub] = cacheEntry{
		perms:     perms,
		expiresAt: time.Now().Add(s.ttl),
	}
	s.mu.Unlock()

	return perms, nil
}
