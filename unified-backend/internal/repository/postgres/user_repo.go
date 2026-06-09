package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unified-backend/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) GetBySub(ctx context.Context, sub string) (*domain.User, error) {
	const q = `
		SELECT id, sub, username, email, role_text, role_id,
		       COALESCE(shlink_api_key, ''), COALESCE(slug_prefix, ''),
		       COALESCE(allowed_domains, ''), status, created_at, updated_at
		FROM users WHERE sub = $1`
	u := &domain.User{}
	var allowedDomainsStr string
	err := r.pool.QueryRow(ctx, q, sub).Scan(
		&u.ID, &u.Sub, &u.Username, &u.Email,
		&u.Role, &u.RoleID, &u.ShlinkAPIKey, &u.SlugPrefix,
		&allowedDomainsStr, &u.Status, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.AllowedDomains = allowedDomainsStr
	return u, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	const q = `
		SELECT id, sub, username, email, role_text, role_id,
		       COALESCE(shlink_api_key, ''), COALESCE(slug_prefix, ''),
		       COALESCE(allowed_domains, ''), status, created_at, updated_at
		FROM users WHERE id = $1`
	u := &domain.User{}
	var allowedDomainsStr string
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&u.ID, &u.Sub, &u.Username, &u.Email,
		&u.Role, &u.RoleID, &u.ShlinkAPIKey, &u.SlugPrefix,
		&allowedDomainsStr, &u.Status, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.AllowedDomains = allowedDomainsStr
	return u, nil
}

func (r *UserRepository) ListAll(ctx context.Context) ([]*domain.User, error) {
	const q = `
		SELECT id, sub, username, email, role_text, role_id,
		       COALESCE(shlink_api_key, ''), COALESCE(slug_prefix, ''),
		       COALESCE(allowed_domains, ''), status, created_at, updated_at
		FROM users ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []*domain.User
	for rows.Next() {
		u := &domain.User{}
		var allowedDomainsStr string
		if err := rows.Scan(
			&u.ID, &u.Sub, &u.Username, &u.Email,
			&u.Role, &u.RoleID, &u.ShlinkAPIKey, &u.SlugPrefix,
			&allowedDomainsStr, &u.Status, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, err
		}
		u.AllowedDomains = allowedDomainsStr
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *UserRepository) Upsert(ctx context.Context, u *domain.User) error {
	const q = `
		INSERT INTO users (sub, username, email, role_text, shlink_api_key, status, slug_prefix, allowed_domains)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (sub) DO UPDATE SET
			username   = EXCLUDED.username,
			email      = EXCLUDED.email,
			slug_prefix = EXCLUDED.slug_prefix,
			allowed_domains = EXCLUDED.allowed_domains,
			updated_at = NOW()
		`
	_, err := r.pool.Exec(
		ctx, q,
		u.Sub, u.Username, u.Email, u.Role, u.ShlinkAPIKey, u.Status,
		u.SlugPrefix, u.AllowedDomains,
	)
	return err
}

func (r *UserRepository) GetRoleIDByName(ctx context.Context, roleName string) (*uuid.UUID, error) {
	const q = `SELECT id FROM roles WHERE name = $1 LIMIT 1`
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, q, roleName).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func (r *UserRepository) UpdateBySubFields(ctx context.Context, sub string, fields map[string]any) error {
	allowed := []string{"role_id", "role_text", "status", "slug_prefix", "shlink_api_key", "username", "email", "allowed_domains"}
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
		return nil
	}
	args = append(args, sub)
	query := fmt.Sprintf(
		"UPDATE users SET %s, updated_at = NOW() WHERE sub = $%d",
		strings.Join(setClauses, ", "), argIdx,
	)
	_, err := r.pool.Exec(ctx, query, args...)
	return err
}

func (r *UserRepository) Pool() *pgxpool.Pool {
	return r.pool
}

