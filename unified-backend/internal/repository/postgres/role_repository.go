package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"unified-backend/internal/domain"
)

type RoleRepository struct {
	pool *pgxpool.Pool
}

func NewRoleRepository(pool *pgxpool.Pool) *RoleRepository {
	return &RoleRepository{pool: pool}
}

func (r *RoleRepository) GetAll(ctx context.Context) ([]domain.RoleEntity, error) {
	const q = `SELECT id, name, COALESCE(description, '') FROM roles ORDER BY name`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []domain.RoleEntity
	for rows.Next() {
		var ro domain.RoleEntity
		if err := rows.Scan(&ro.ID, &ro.Name, &ro.Description); err != nil {
			return nil, err
		}
		roles = append(roles, ro)
	}
	return roles, rows.Err()
}

func (r *RoleRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.RoleEntity, error) {
	const q = `SELECT id, name, COALESCE(description, '') FROM roles WHERE id = $1`
	ro := &domain.RoleEntity{}
	err := r.pool.QueryRow(ctx, q, id).Scan(&ro.ID, &ro.Name, &ro.Description)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return ro, err
}

func (r *RoleRepository) GetByName(ctx context.Context, name string) (*domain.RoleEntity, error) {
	const q = `SELECT id, name, COALESCE(description, '') FROM roles WHERE name = $1`
	ro := &domain.RoleEntity{}
	err := r.pool.QueryRow(ctx, q, name).Scan(&ro.ID, &ro.Name, &ro.Description)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return ro, err
}

func (r *RoleRepository) Create(ctx context.Context, name, description string) (*domain.RoleEntity, error) {
	const q = `
		INSERT INTO roles (id, name, description)
		VALUES (gen_random_uuid(), $1, $2)
		RETURNING id, name, COALESCE(description, '')`
	ro := &domain.RoleEntity{}
	err := r.pool.QueryRow(ctx, q, name, description).Scan(&ro.ID, &ro.Name, &ro.Description)
	return ro, err
}

// GetPermissions возвращает все разрешения роли.
func (r *RoleRepository) GetPermissions(ctx context.Context, roleID uuid.UUID) ([]domain.Permission, error) {
	const q = `
		SELECT p.id, p.name, COALESCE(p.description, '')
		FROM permissions p
		JOIN role_permissions rp ON rp.permission_id = p.id
		WHERE rp.role_id = $1
		ORDER BY p.name`
	rows, err := r.pool.Query(ctx, q, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []domain.Permission
	for rows.Next() {
		var p domain.Permission
		if err := rows.Scan(&p.ID, &p.Name, &p.Description); err != nil {
			return nil, err
		}
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

// SetPermissions заменяет весь набор разрешений роли (DELETE + INSERT).
func (r *RoleRepository) SetPermissions(ctx context.Context, roleID uuid.UUID, permissionIDs []uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx, `DELETE FROM role_permissions WHERE role_id = $1`, roleID)
	if err != nil {
		return err
	}

	for _, pid := range permissionIDs {
		_, err = tx.Exec(ctx,
			`INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			roleID, pid,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
