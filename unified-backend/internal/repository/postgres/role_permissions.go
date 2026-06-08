package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"unified-backend/internal/domain"
)

type RolePermissionsRepository struct {
	pool *pgxpool.Pool
}

func NewRolePermissionsRepository(pool *pgxpool.Pool) *RolePermissionsRepository {
	return &RolePermissionsRepository{pool: pool}
}

const selectPermissionsSQL = `
SELECT role,
    can_view_own_links, can_view_all_links,
    can_create_links, can_create_with_custom_slug, can_create_without_slug,
    can_edit_own_links, can_edit_all_links,
    can_delete_own_links, can_delete_all_links,
    can_manage_own_tags, can_manage_all_tags,
    can_view_own_stats, can_view_all_stats,
    can_view_audit_logs, can_manage_users, can_manage_roles,
    updated_at
FROM role_permissions
`

// GetAll возвращает все строки role_permissions.
func (r *RolePermissionsRepository) GetAll(ctx context.Context) ([]domain.RolePermissions, error) {
	rows, err := r.pool.Query(ctx, selectPermissionsSQL)
	if err != nil {
		return nil, fmt.Errorf("role_permissions GetAll: %w", err)
	}
	defer rows.Close()

	var result []domain.RolePermissions
	for rows.Next() {
		var p domain.RolePermissions
		if err := rows.Scan(
			&p.Role,
			&p.CanViewOwnLinks, &p.CanViewAllLinks,
			&p.CanCreateLinks, &p.CanCreateWithCustomSlug, &p.CanCreateWithoutSlug,
			&p.CanEditOwnLinks, &p.CanEditAllLinks,
			&p.CanDeleteOwnLinks, &p.CanDeleteAllLinks,
			&p.CanManageOwnTags, &p.CanManageAllTags,
			&p.CanViewOwnStats, &p.CanViewAllStats,
			&p.CanViewAuditLogs, &p.CanManageUsers, &p.CanManageRoles,
			&p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("role_permissions scan: %w", err)
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

// GetByRole возвращает permissions для конкретной роли.
func (r *RolePermissionsRepository) GetByRole(ctx context.Context, role string) (*domain.RolePermissions, error) {
	query := selectPermissionsSQL + ` WHERE role = $1`
	row := r.pool.QueryRow(ctx, query, role)

	var p domain.RolePermissions
	err := row.Scan(
		&p.Role,
		&p.CanViewOwnLinks, &p.CanViewAllLinks,
		&p.CanCreateLinks, &p.CanCreateWithCustomSlug, &p.CanCreateWithoutSlug,
		&p.CanEditOwnLinks, &p.CanEditAllLinks,
		&p.CanDeleteOwnLinks, &p.CanDeleteAllLinks,
		&p.CanManageOwnTags, &p.CanManageAllTags,
		&p.CanViewOwnStats, &p.CanViewAllStats,
		&p.CanViewAuditLogs, &p.CanManageUsers, &p.CanManageRoles,
		&p.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("role_permissions GetByRole: %w", err)
	}
	return &p, nil
}

// Upsert создаёт или полностью обновляет permissions для роли.
func (r *RolePermissionsRepository) Upsert(ctx context.Context, p *domain.RolePermissions) error {
	_, err := r.pool.Exec(ctx, `
INSERT INTO role_permissions (
    role,
    can_view_own_links, can_view_all_links,
    can_create_links, can_create_with_custom_slug, can_create_without_slug,
    can_edit_own_links, can_edit_all_links,
    can_delete_own_links, can_delete_all_links,
    can_manage_own_tags, can_manage_all_tags,
    can_view_own_stats, can_view_all_stats,
    can_view_audit_logs, can_manage_users, can_manage_roles,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, NOW()
)
ON CONFLICT (role) DO UPDATE SET
    can_view_own_links          = EXCLUDED.can_view_own_links,
    can_view_all_links          = EXCLUDED.can_view_all_links,
    can_create_links            = EXCLUDED.can_create_links,
    can_create_with_custom_slug = EXCLUDED.can_create_with_custom_slug,
    can_create_without_slug     = EXCLUDED.can_create_without_slug,
    can_edit_own_links          = EXCLUDED.can_edit_own_links,
    can_edit_all_links          = EXCLUDED.can_edit_all_links,
    can_delete_own_links        = EXCLUDED.can_delete_own_links,
    can_delete_all_links        = EXCLUDED.can_delete_all_links,
    can_manage_own_tags         = EXCLUDED.can_manage_own_tags,
    can_manage_all_tags         = EXCLUDED.can_manage_all_tags,
    can_view_own_stats          = EXCLUDED.can_view_own_stats,
    can_view_all_stats          = EXCLUDED.can_view_all_stats,
    can_view_audit_logs         = EXCLUDED.can_view_audit_logs,
    can_manage_users            = EXCLUDED.can_manage_users,
    can_manage_roles            = EXCLUDED.can_manage_roles,
    updated_at                  = NOW()
`,
		p.Role,
		p.CanViewOwnLinks, p.CanViewAllLinks,
		p.CanCreateLinks, p.CanCreateWithCustomSlug, p.CanCreateWithoutSlug,
		p.CanEditOwnLinks, p.CanEditAllLinks,
		p.CanDeleteOwnLinks, p.CanDeleteAllLinks,
		p.CanManageOwnTags, p.CanManageAllTags,
		p.CanViewOwnStats, p.CanViewAllStats,
		p.CanViewAuditLogs, p.CanManageUsers, p.CanManageRoles,
	)
	if err != nil {
		return fmt.Errorf("role_permissions Upsert: %w", err)
	}
	return nil
}

// Delete удаляет роль из role_permissions. Нет строки — не ошибка.
func (r *RolePermissionsRepository) Delete(ctx context.Context, role string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM role_permissions WHERE role = $1`, role)
	if err != nil {
		return fmt.Errorf("role_permissions Delete: %w", err)
	}
	return nil
}
