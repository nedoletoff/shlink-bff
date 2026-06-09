package service_test

import (
	"context"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"unified-backend/internal/service"
)

// получаем DSN из окружения, иначе скип аютентичных тестов
	func dsn() string {
	return os.Getenv("TEST_DATABASE_URL")
}

const schema = `
CREATE TABLE IF NOT EXISTS users (
	sub         TEXT PRIMARY KEY,
	email       TEXT NOT NULL DEFAULT '',
	name        TEXT NOT NULL DEFAULT '',
	role        TEXT NOT NULL DEFAULT 'user',
	is_active   BOOLEAN NOT NULL DEFAULT true,
	created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	role_id     UUID
);
CREATE TABLE IF NOT EXISTS roles (
	id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	name        TEXT UNIQUE NOT NULL,
	description TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS permissions (
	id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	name        TEXT UNIQUE NOT NULL,
	description TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS role_permissions_v2 (
	role_id       UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
	permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
	PRIMARY KEY (role_id, permission_id)
);
`

func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	d := dsn()
	if d == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, d)
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close() })

	_, err = pool.Exec(ctx, schema)
	require.NoError(t, err)

	// truncate for isolation
	_, err = pool.Exec(ctx, `TRUNCATE role_permissions_v2, permissions, roles, users RESTART IDENTITY CASCADE`)
	require.NoError(t, err)

	return pool
}

func insertRole(t *testing.T, pool *pgxpool.Pool, name string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := pool.QueryRow(context.Background(),
		`INSERT INTO roles (id, name) VALUES (gen_random_uuid(), $1) RETURNING id`, name,
	).Scan(&id)
	require.NoError(t, err)
	return id
}

func insertPerm(t *testing.T, pool *pgxpool.Pool, name string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := pool.QueryRow(context.Background(),
		`INSERT INTO permissions (id, name) VALUES (gen_random_uuid(), $1) RETURNING id`, name,
	).Scan(&id)
	require.NoError(t, err)
	return id
}

func assignPerm(t *testing.T, pool *pgxpool.Pool, roleID, permID uuid.UUID) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO role_permissions_v2 (role_id, permission_id) VALUES ($1, $2)`, roleID, permID)
	require.NoError(t, err)
}

func insertUser(t *testing.T, pool *pgxpool.Pool, sub string, roleID *uuid.UUID) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO users (sub, email, name, role_id) VALUES ($1, $2, $3, $4)`,
		sub, sub+"@test", sub, roleID,
	)
	require.NoError(t, err)
}

// ───────────────────────────────────────────────────────────────────────────────
func TestUserHasPermission(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	adminRole    := insertRole(t, pool, "admin")
	viewerRole   := insertRole(t, pool, "viewer")
	auditorRole  := insertRole(t, pool, "auditor_admin")

	pCreate := insertPerm(t, pool, "short_urls.create")
	pView   := insertPerm(t, pool, "dashboard.view")

	assignPerm(t, pool, adminRole, pCreate)
	assignPerm(t, pool, adminRole, pView)
	assignPerm(t, pool, viewerRole, pCreate)
	assignPerm(t, pool, auditorRole, pView) // auditor_admin без short_urls.create

	insertUser(t, pool, "admin-user",   &adminRole)
	insertUser(t, pool, "viewer-user",  &viewerRole)
	insertUser(t, pool, "auditor-user", &auditorRole)
	insertUser(t, pool, "norole-user",  nil)

	svc := service.NewPermissionService(pool)

	cases := []struct {
		sub    string
		action string
		want   bool
	}{
		{"admin-user",   "short_urls.create", true},
		{"admin-user",   "dashboard.view",    true},
		{"viewer-user",  "short_urls.create", true},
		{"viewer-user",  "dashboard.view",    false},
		{"auditor-user", "short_urls.create", false}, // ключевой критерий
		{"auditor-user", "dashboard.view",    true},
		{"norole-user",  "short_urls.create", false},
	}
	for _, tc := range cases {
		t.Run(tc.sub+":"+tc.action, func(t *testing.T) {
			got, err := svc.UserHasPermission(ctx, tc.sub, tc.action)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGetUserPermissions(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	role := insertRole(t, pool, "editor")
	p1   := insertPerm(t, pool, "short_urls.create")
	p2   := insertPerm(t, pool, "short_urls.delete")
	p3   := insertPerm(t, pool, "dashboard.view")
	assignPerm(t, pool, role, p1)
	assignPerm(t, pool, role, p2)
	assignPerm(t, pool, role, p3)
	insertUser(t, pool, "editor-user", &role)

	svc := service.NewPermissionService(pool)
	perms, err := svc.GetUserPermissions(ctx, "editor-user")
	require.NoError(t, err)
	sort.Strings(perms)
	assert.Equal(t, []string{"dashboard.view", "short_urls.create", "short_urls.delete"}, perms)
}

func TestCacheHit(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	role := insertRole(t, pool, "cached_role")
	perm := insertPerm(t, pool, "some.action")
	assignPerm(t, pool, role, perm)
	insertUser(t, pool, "cached-user", &role)

	svc := service.NewPermissionService(pool)

	// первый запрос — в БД
	got1, err := svc.UserHasPermission(ctx, "cached-user", "some.action")
	require.NoError(t, err)
	assert.True(t, got1)

	// удаляем разрешение напрямую в БД, но кэш ещё жив
	_, _ = pool.Exec(ctx, `DELETE FROM role_permissions_v2 WHERE role_id=$1`, role)

	got2, err := svc.UserHasPermission(ctx, "cached-user", "some.action")
	require.NoError(t, err)
	assert.True(t, got2, "должно вернуть значение из кэша")
}

func TestCacheInvalidation(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	role1 := insertRole(t, pool, "role_a")
	role2 := insertRole(t, pool, "role_b")
	perm  := insertPerm(t, pool, "special.action")
	assignPerm(t, pool, role1, perm) // role_a имеет special.action
	// role_b — нет
	insertUser(t, pool, "switch-user", &role1)

	svc := service.NewPermissionService(pool)

	got, err := svc.UserHasPermission(ctx, "switch-user", "special.action")
	require.NoError(t, err)
	assert.True(t, got)

	// меняем роль пользователя на role_b (PATCH /users/:sub/role)
	_, err = pool.Exec(ctx, `UPDATE users SET role_id=$1 WHERE sub=$2`, role2, "switch-user")
	require.NoError(t, err)

	// инвалидация кэша
	svc.InvalidateUser("switch-user")

	got2, err := svc.UserHasPermission(ctx, "switch-user", "special.action")
	require.NoError(t, err)
	assert.False(t, got2, "после смены роли должно вернуть false")
}

func TestCacheTTLExpiry(t *testing.T) {
	if testing.Short() {
		t.Skip("пропускаем в -short режиме")
	}
	pool := setupTestDB(t)
	ctx := context.Background()

	role := insertRole(t, pool, "ttl_role")
	perm := insertPerm(t, pool, "ttl.action")
	assignPerm(t, pool, role, perm)
	insertUser(t, pool, "ttl-user", &role)

	// TTL = 1s для теста (PermissionService принимает функциональные опции)
	svc := service.NewPermissionService(pool, service.WithCacheTTL(1*time.Second))

	got, err := svc.UserHasPermission(ctx, "ttl-user", "ttl.action")
	require.NoError(t, err)
	assert.True(t, got)

	_, _ = pool.Exec(ctx, `DELETE FROM role_permissions_v2 WHERE role_id=$1`, role)

	// до истечения TTL — кэш ещё живёт
	got2, _ := svc.UserHasPermission(ctx, "ttl-user", "ttl.action")
	assert.True(t, got2)

	time.Sleep(1100 * time.Millisecond)

	// TTL истёк — идём в БД
	got3, err := svc.UserHasPermission(ctx, "ttl-user", "ttl.action")
	require.NoError(t, err)
	assert.False(t, got3, "после TTL должно вернуть false")
}
