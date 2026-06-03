package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"unified-backend/internal/domain"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) GetBySub(ctx context.Context, sub string) (*domain.User, error) {
	const q = `
		SELECT id, sub, username, email, role, shlink_api_key,
		       COALESCE(slug_prefix, ''), status, created_at, updated_at
		FROM users WHERE sub = $1`

	u := &domain.User{}
	err := r.pool.QueryRow(ctx, q, sub).Scan(
		&u.ID, &u.Sub, &u.Username, &u.Email,
		&u.Role, &u.ShlinkAPIKey, &u.SlugPrefix,
		&u.Status, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return u, err
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	const q = `
		SELECT id, sub, username, email, role, shlink_api_key,
		       COALESCE(slug_prefix, ''), status, created_at, updated_at
		FROM users WHERE id = $1`

	u := &domain.User{}
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&u.ID, &u.Sub, &u.Username, &u.Email,
		&u.Role, &u.ShlinkAPIKey, &u.SlugPrefix,
		&u.Status, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return u, err
}

func (r *UserRepository) ListAll(ctx context.Context) ([]*domain.User, error) {
	const q = `
		SELECT id, sub, username, email, role, shlink_api_key,
		       COALESCE(slug_prefix, ''), status, created_at, updated_at
		FROM users ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		u := &domain.User{}
		if err := rows.Scan(
			&u.ID, &u.Sub, &u.Username, &u.Email,
			&u.Role, &u.ShlinkAPIKey, &u.SlugPrefix,
			&u.Status, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// Upsert вставляет пользователя или обновляет username/email при повторном логине.
//
// ВАЖНО (#32): role НЕ обновляется при повторном логине намеренно — роль в БД
// является источником истины после первого провизионирования, чтобы админ мог
// понижать/повышать роль вручную, а Keycloak не затирал это при каждом входе.
// См. раздел README "Управление ролями".
func (r *UserRepository) Upsert(ctx context.Context, u *domain.User) error {
	const q = `
		INSERT INTO users (sub, username, email, role, shlink_api_key, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (sub) DO UPDATE SET
			username   = EXCLUDED.username,
			email      = EXCLUDED.email,
			-- role НЕ обновляется намеренно (#32)
			updated_at = NOW()`
	_, err := r.pool.Exec(ctx, q,
		u.Sub, u.Username, u.Email, u.Role, u.ShlinkAPIKey, u.Status,
	)
	return err
}

// UpdateBySubFields обновляет разрешённые поля пользователя ОДНИМ атомарным UPDATE (#14).
//
// Прежняя реализация выполняла до трёх отдельных UPDATE без транзакции,
// что давало inconsistent-состояние между запросами.
func (r *UserRepository) UpdateBySubFields(ctx context.Context, sub string, fields map[string]any) error {
	// Белый список колонок — защита от SQL-инъекции в имена полей.
	allowed := []string{"role", "status", "slug_prefix"}

	setClauses := make([]string, 0, len(allowed))
	args := make([]any, 0, len(allowed)+1)
	argIdx := 1
	for _, col := range allowed {
		if v, ok := fields[col]; ok {
			setClauses = append(setClauses, fmt.Sprintf("%s = $%d", col, argIdx))
			args = append(args, v)
			argIdx++
		}
	}

	if len(setClauses) == 0 {
		return nil // нечего обновлять
	}

	args = append(args, sub)
	query := fmt.Sprintf(
		"UPDATE users SET %s, updated_at = NOW() WHERE sub = $%d",
		strings.Join(setClauses, ", "), argIdx,
	)
	_, err := r.pool.Exec(ctx, query, args...)
	return err
}

func (r *UserRepository) UpdateAPIKey(ctx context.Context, sub, newKey string) error {
	const q = `UPDATE users SET shlink_api_key = $1, updated_at = NOW() WHERE sub = $2`
	_, err := r.pool.Exec(ctx, q, newKey, sub)
	return err
}

func (r *UserRepository) UpdateSlugPrefix(ctx context.Context, sub, prefix string) error {
	const q = `UPDATE users SET slug_prefix = $1, updated_at = NOW() WHERE sub = $2`
	_, err := r.pool.Exec(ctx, q, prefix, sub)
	return err
}
