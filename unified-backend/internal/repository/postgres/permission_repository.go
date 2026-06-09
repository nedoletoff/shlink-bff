package postgres

import (
	"context"
	"errors"
	"unified-backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PermissionRepository struct {
	pool *pgxpool.Pool
}

func NewPermissionRepository(pool *pgxpool.Pool) *PermissionRepository {
	return &PermissionRepository{pool: pool}
}

func (r *PermissionRepository) GetAll(ctx context.Context) ([]domain.Permission, error) {
	const q = `SELECT id, name, COALESCE(description, '') FROM permissions ORDER BY name`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Permission
	for rows.Next() {
		var p domain.Permission
		if err := rows.Scan(&p.ID, &p.Name, &p.Description); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (r *PermissionRepository) GetByName(ctx context.Context, name string) (*domain.Permission, error) {
	const q = `SELECT id, name, COALESCE(description, '') FROM permissions WHERE name = $1`
	p := &domain.Permission{}
	err := r.pool.QueryRow(ctx, q, name).Scan(&p.ID, &p.Name, &p.Description)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return p, err
}
