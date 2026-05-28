package postgres

import (
	"context"
	"errors"
	"time"

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

func (r *UserRepository) Upsert(ctx context.Context, u *domain.User) error {
	const q = `
		INSERT INTO users (sub, username, email, role, shlink_api_key, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (sub) DO UPDATE SET
			username   = EXCLUDED.username,
			email      = EXCLUDED.email,
			updated_at = NOW()`
	_, err := r.pool.Exec(ctx, q,
		u.Sub, u.Username, u.Email, u.Role, u.ShlinkAPIKey, u.Status,
	)
	return err
}

func (r *UserRepository) UpdateBySubFields(ctx context.Context, sub string, fields map[string]any) error {
	// Обновляем только разрешённые поля
	allowed := map[string]bool{
		"role": true, "status": true, "slug_prefix": true,
	}
	for k := range fields {
		if !allowed[k] {
			delete(fields, k)
		}
	}
	fields["updated_at"] = time.Now()

	// Простое поле-за-полем (для production стоит использовать squirrel/bun)
	if v, ok := fields["role"]; ok {
		if _, err := r.pool.Exec(ctx, `UPDATE users SET role=$1, updated_at=NOW() WHERE sub=$2`, v, sub); err != nil {
			return err
		}
	}
	if v, ok := fields["status"]; ok {
		if _, err := r.pool.Exec(ctx, `UPDATE users SET status=$1, updated_at=NOW() WHERE sub=$2`, v, sub); err != nil {
			return err
		}
	}
	if v, ok := fields["slug_prefix"]; ok {
		if _, err := r.pool.Exec(ctx, `UPDATE users SET slug_prefix=$1, updated_at=NOW() WHERE sub=$2`, v, sub); err != nil {
			return err
		}
	}
	return nil
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
