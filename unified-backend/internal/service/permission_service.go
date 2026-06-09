package service

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultCacheTTL = 5 * time.Minute

type cacheEntry struct {
	perms     []string
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

// UserHasPermission проверяет наличие разрешения у пользователя по userID.
func (s *PermissionService) UserHasPermission(ctx context.Context, userID uuid.UUID, action string) (bool, error) {
	// сначала получаем sub по userID
	var sub string
	err := s.db.QueryRow(ctx, `SELECT sub FROM users WHERE id = $1`, userID).Scan(&sub)
	if err != nil {
		return false, err
	}
	return s.UserHasPermissionBySub(ctx, sub, action)
}

// UserHasPermissionBySub проверяет наличие разрешения по subject.
func (s *PermissionService) UserHasPermissionBySub(ctx context.Context, sub string, action string) (bool, error) {
	perms, err := s.GetUserPermissions(ctx, sub)
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
func (s *PermissionService) InvalidateUser(userID uuid.UUID) {
	var sub string
	err := s.db.QueryRow(context.Background(), `SELECT sub FROM users WHERE id = $1`, userID).Scan(&sub)
	if err != nil {
		// если не нашли – просто возвращаемся
		return
	}
	s.mu.Lock()
	delete(s.cache, sub)
	s.mu.Unlock()
}

// InvalidateRole сбрасывает кэш всех пользователей, имеющих указанную роль.
// Для простоты сбрасываем весь кэш.
func (s *PermissionService) InvalidateRole(roleName string) {
	s.mu.Lock()
	s.cache = make(map[string]cacheEntry)
	s.mu.Unlock()
}

// InvalidateAll очищает весь кэш.
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

