package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"unified-backend/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type URLOwnershipRepository struct {
	pool *pgxpool.Pool
}

func NewURLOwnershipRepository(pool *pgxpool.Pool) *URLOwnershipRepository {
	return &URLOwnershipRepository{pool: pool}
}

// Save – сохраняет или обновляет запись с метаданными
func (r *URLOwnershipRepository) Save(ctx context.Context, shortCode, ownerSub, ownerUsername, domain string, metadata *domain.ShortURLMetadata) error {
	query := `
		INSERT INTO url_ownership (
			short_code, domain, owner_sub, owner_username,
			title, is_active, valid_since, valid_until, max_visits, is_public, tags,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
		ON CONFLICT (short_code, domain) DO UPDATE SET
			owner_sub = EXCLUDED.owner_sub,
			owner_username = EXCLUDED.owner_username,
			title = EXCLUDED.title,
			is_active = EXCLUDED.is_active,
			valid_since = EXCLUDED.valid_since,
			valid_until = EXCLUDED.valid_until,
			max_visits = EXCLUDED.max_visits,
			is_public = EXCLUDED.is_public,
			tags = EXCLUDED.tags,
			deactivated_at = NULL,
			deactivated_by = NULL,
			deleted_at = NULL,
			deleted_by = NULL
	`
	_, err := r.pool.Exec(
		ctx, query,
		shortCode, domain, ownerSub, ownerUsername,
		metadata.Title, metadata.IsActive, metadata.ValidSince, metadata.ValidUntil,
		metadata.MaxVisits, metadata.IsPublic, metadata.Tags,
	)
	return err
}

// GetOwnership – возвращает метаданные ссылки
func (r *URLOwnershipRepository) GetOwnership(ctx context.Context, shortCode, domain string) (*domain.ShortURLMetadata, error) {
	var rec domain.ShortURLMetadata
	var validSince, validUntil sql.NullTime
	var tagsJSON []byte
	var title sql.NullString
	var deactivatedAt, deletedAt sql.NullTime
	var deactivatedBy, deletedBy sql.NullString

	err := r.pool.QueryRow(
		ctx,
		`SELECT short_code, domain, owner_sub, owner_username,
			COALESCE(title, ''),
			is_active, valid_since, valid_until, max_visits, is_public, tags,
			deactivated_at, deactivated_by, created_at, deleted_at, deleted_by
		 FROM url_ownership
		 WHERE short_code = $1 AND domain = $2 AND deleted_at IS NULL`,
		shortCode, domain,
	).Scan(
		&rec.ShortCode, &rec.Domain, &rec.OwnerSub, &rec.OwnerUsername,
		&title, &rec.IsActive, &validSince, &validUntil, &rec.MaxVisits,
		&rec.IsPublic, &tagsJSON, &deactivatedAt, &deactivatedBy,
		&rec.CreatedAt, &deletedAt, &deletedBy,
	)
	if err != nil {
		return nil, err
	}
	rec.Title = title.String
	if validSince.Valid {
		rec.ValidSince = &validSince.Time
	}
	if validUntil.Valid {
		rec.ValidUntil = &validUntil.Time
	}
	if tagsJSON != nil {
		_ = json.Unmarshal(tagsJSON, &rec.Tags)
	}
	if deactivatedAt.Valid {
		rec.DeactivatedAt = &deactivatedAt.Time
	}
	if deactivatedBy.Valid {
		rec.DeactivatedBy = &deactivatedBy.String
	}
	if deletedAt.Valid {
		rec.DeletedAt = &deletedAt.Time
	}
	if deletedBy.Valid {
		rec.DeletedBy = &deletedBy.String
	}
	return &rec, nil
}

// GetBatch – массовое получение метаданных по списку shortCode
func (r *URLOwnershipRepository) GetBatch(ctx context.Context, shortCodes []string, domain string) (map[string]*domain.ShortURLMetadata, error) {
	if len(shortCodes) == 0 {
		return make(map[string]*domain.ShortURLMetadata), nil
	}
	rows, err := r.pool.Query(
		ctx,
		`SELECT short_code, domain, owner_sub, owner_username,
			COALESCE(title, ''),
			is_active, valid_since, valid_until, max_visits, is_public, tags,
			deactivated_at, deactivated_by, created_at, deleted_at, deleted_by
		 FROM url_ownership
		 WHERE short_code = ANY($1) AND domain = $2 AND deleted_at IS NULL`,
		shortCodes, domain,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]*domain.ShortURLMetadata, len(shortCodes))
	for rows.Next() {
		var rec domain.ShortURLMetadata
		var validSince, validUntil sql.NullTime
		var tagsJSON []byte
		var title sql.NullString
		var deactivatedAt, deletedAt sql.NullTime
		var deactivatedBy, deletedBy sql.NullString
		err := rows.Scan(
			&rec.ShortCode, &rec.Domain, &rec.OwnerSub, &rec.OwnerUsername,
			&title, &rec.IsActive, &validSince, &validUntil, &rec.MaxVisits,
			&rec.IsPublic, &tagsJSON, &deactivatedAt, &deactivatedBy,
			&rec.CreatedAt, &deletedAt, &deletedBy,
		)
		if err != nil {
			return nil, err
		}
		rec.Title = title.String
		if validSince.Valid {
			rec.ValidSince = &validSince.Time
		}
		if validUntil.Valid {
			rec.ValidUntil = &validUntil.Time
		}
		if tagsJSON != nil {
			_ = json.Unmarshal(tagsJSON, &rec.Tags)
		}
		if deactivatedAt.Valid {
			rec.DeactivatedAt = &deactivatedAt.Time
		}
		if deactivatedBy.Valid {
			rec.DeactivatedBy = &deactivatedBy.String
		}
		if deletedAt.Valid {
			rec.DeletedAt = &deletedAt.Time
		}
		if deletedBy.Valid {
			rec.DeletedBy = &deletedBy.String
		}
		result[rec.ShortCode] = &rec
	}
	return result, rows.Err()
}

// GetAllByOwner – все ссылки владельца (активные и деактивированные, но не удалённые)
func (r *URLOwnershipRepository) GetAllByOwner(ctx context.Context, ownerSub string) ([]*domain.ShortURLMetadata, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT short_code, domain, owner_sub, owner_username,
			COALESCE(title, ''),
			is_active, valid_since, valid_until, max_visits, is_public, tags,
			deactivated_at, deactivated_by, created_at, deleted_at, deleted_by
		 FROM url_ownership
		 WHERE owner_sub = $1 AND deleted_at IS NULL
		 ORDER BY created_at DESC`,
		ownerSub,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.ShortURLMetadata
	for rows.Next() {
		var rec domain.ShortURLMetadata
		var validSince, validUntil sql.NullTime
		var tagsJSON []byte
		var title sql.NullString
		var deactivatedAt, deletedAt sql.NullTime
		var deactivatedBy, deletedBy sql.NullString
		err := rows.Scan(
			&rec.ShortCode, &rec.Domain, &rec.OwnerSub, &rec.OwnerUsername,
			&title, &rec.IsActive, &validSince, &validUntil, &rec.MaxVisits,
			&rec.IsPublic, &tagsJSON, &deactivatedAt, &deactivatedBy,
			&rec.CreatedAt, &deletedAt, &deletedBy,
		)
		if err != nil {
			return nil, err
		}
		rec.Title = title.String
		if validSince.Valid {
			rec.ValidSince = &validSince.Time
		}
		if validUntil.Valid {
			rec.ValidUntil = &validUntil.Time
		}
		if tagsJSON != nil {
			_ = json.Unmarshal(tagsJSON, &rec.Tags)
		}
		if deactivatedAt.Valid {
			rec.DeactivatedAt = &deactivatedAt.Time
		}
		if deactivatedBy.Valid {
			rec.DeactivatedBy = &deactivatedBy.String
		}
		if deletedAt.Valid {
			rec.DeletedAt = &deletedAt.Time
		}
		if deletedBy.Valid {
			rec.DeletedBy = &deletedBy.String
		}
		result = append(result, &rec)
	}
	return result, rows.Err()
}

// IsOwner проверяет владение активной (не удалённой) ссылкой.
func (r *URLOwnershipRepository) IsOwner(ctx context.Context, shortCode, domain, sub string) (bool, error) {
	var count int
	err := r.pool.QueryRow(
		ctx,
		`SELECT COUNT(*) FROM url_ownership
		 WHERE short_code = $1 AND domain = $2 AND owner_sub = $3
		   AND deleted_at IS NULL`,
		shortCode, domain, sub,
	).Scan(&count)
	return count > 0, err
}

// Deactivate устанавливает is_active=false, заполняет deactivated_at/by.
func (r *URLOwnershipRepository) Deactivate(ctx context.Context, shortCode, domain, actorSub string) error {
	_, err := r.pool.Exec(
		ctx,
		`UPDATE url_ownership
		 SET is_active = FALSE,
		     deactivated_at = COALESCE(deactivated_at, NOW()),
		     deactivated_by = COALESCE(deactivated_by, $3)
		 WHERE short_code = $1 AND domain = $2 AND deleted_at IS NULL`,
		shortCode, domain, actorSub,
	)
	return err
}

// Activate сбрасывает деактивацию.
func (r *URLOwnershipRepository) Activate(ctx context.Context, shortCode, domain string) error {
	_, err := r.pool.Exec(
		ctx,
		`UPDATE url_ownership
		 SET is_active = TRUE,
		     deactivated_at = NULL,
		     deactivated_by = NULL
		 WHERE short_code = $1 AND domain = $2 AND deleted_at IS NULL`,
		shortCode, domain,
	)
	return err
}

// SoftDelete помечает запись как удалённую (логическое удаление).
func (r *URLOwnershipRepository) SoftDelete(ctx context.Context, shortCode, domain, actorSub string) error {
	_, err := r.pool.Exec(
		ctx,
		`UPDATE url_ownership
		 SET deleted_at = NOW(),
		     deleted_by = $3,
		     is_active = FALSE
		 WHERE short_code = $1 AND domain = $2 AND deleted_at IS NULL`,
		shortCode, domain, actorSub,
	)
	return err
}

// HardDelete физически удаляет запись.
func (r *URLOwnershipRepository) HardDelete(ctx context.Context, shortCode, domain string) error {
	_, err := r.pool.Exec(
		ctx,
		`DELETE FROM url_ownership WHERE short_code = $1 AND domain = $2`,
		shortCode, domain,
	)
	return err
}

// GetShortCodeSet возвращает set всех short_code владельца (не удалённых).
func (r *URLOwnershipRepository) GetShortCodeSet(ctx context.Context, ownerSub string) (map[string]struct{}, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT short_code FROM url_ownership
		 WHERE owner_sub = $1 AND deleted_at IS NULL`,
		ownerSub,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	set := make(map[string]struct{})
	for rows.Next() {
		var sc string
		if err := rows.Scan(&sc); err != nil {
			return nil, err
		}
		set[sc] = struct{}{}
	}
	return set, rows.Err()
}

// GetActiveCodeSet возвращает set только активных (is_active=true) ссылок владельца.
func (r *URLOwnershipRepository) GetActiveCodeSet(ctx context.Context, ownerSub string) (map[string]struct{}, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT short_code FROM url_ownership
		 WHERE owner_sub = $1 AND deleted_at IS NULL AND is_active = TRUE`,
		ownerSub,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	set := make(map[string]struct{})
	for rows.Next() {
		var sc string
		if err := rows.Scan(&sc); err != nil {
			return nil, err
		}
		set[sc] = struct{}{}
	}
	return set, rows.Err()
}

// GetStatusCodeSet возвращает map[shortCode]isActive для владельца.
func (r *URLOwnershipRepository) GetStatusCodeSet(ctx context.Context, ownerSub string) (map[string]bool, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT short_code, is_active FROM url_ownership
		 WHERE owner_sub = $1 AND deleted_at IS NULL`,
		ownerSub,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string]bool)
	for rows.Next() {
		var sc string
		var active bool
		if err := rows.Scan(&sc, &active); err != nil {
			return nil, err
		}
		m[sc] = active
	}
	return m, rows.Err()
}

