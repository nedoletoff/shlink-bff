package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// URLOwnershipRecord — запись в url_ownership.
type URLOwnershipRecord struct {
	ShortCode string
	Domain    string
	OwnerSub  string
	CreatedAt time.Time
	DeletedAt *time.Time
	DeletedBy *string
}

// URLOwnershipRepository — хранит и читает ownership коротких ссылок.
type URLOwnershipRepository struct {
	pool *pgxpool.Pool
}

// NewURLOwnershipRepository создаёт новый URLOwnershipRepository.
func NewURLOwnershipRepository(pool *pgxpool.Pool) *URLOwnershipRepository {
	return &URLOwnershipRepository{pool: pool}
}

// Save сохраняет новую запись ownership.
// При конфликте (повторное создание с тем же short_code+domain) — игнорируется (DO NOTHING).
func (r *URLOwnershipRepository) Save(ctx context.Context, shortCode, ownerSub, domain string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO url_ownership (short_code, domain, owner_sub)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (short_code, domain) DO NOTHING`,
		shortCode, domain, ownerSub,
	)
	return err
}

// GetOwner возвращает owner_sub для данного short_code+domain.
func (r *URLOwnershipRepository) GetOwner(ctx context.Context, shortCode, domain string) (string, error) {
	var ownerSub string
	err := r.pool.QueryRow(ctx,
		`SELECT owner_sub FROM url_ownership
		 WHERE short_code = $1 AND domain = $2`,
		shortCode, domain,
	).Scan(&ownerSub)
	if err != nil {
		return "", err
	}
	return ownerSub, nil
}

// IsOwner проверяет является ли sub владельцем ссылки.
// Учитывает только активные (не удалённые) записи.
func (r *URLOwnershipRepository) IsOwner(ctx context.Context, shortCode, domain, sub string) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM url_ownership
		 WHERE short_code = $1 AND domain = $2 AND owner_sub = $3
		   AND deleted_at IS NULL`,
		shortCode, domain, sub,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// SoftDelete помечает ссылку как удалённую (заполняет deleted_at и deleted_by).
func (r *URLOwnershipRepository) SoftDelete(ctx context.Context, shortCode, domain, deletedBy string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE url_ownership
		    SET deleted_at = now(), deleted_by = $3
		  WHERE short_code = $1 AND domain = $2
		    AND deleted_at IS NULL`,
		shortCode, domain, deletedBy,
	)
	return err
}

// GetByOwner возвращает все активные (не удалённые) ссылки пользователя.
func (r *URLOwnershipRepository) GetByOwner(ctx context.Context, ownerSub string) ([]URLOwnershipRecord, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT short_code, domain, owner_sub, created_at, deleted_at, deleted_by
		   FROM url_ownership
		  WHERE owner_sub = $1 AND deleted_at IS NULL
		  ORDER BY created_at DESC`,
		ownerSub,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []URLOwnershipRecord
	for rows.Next() {
		var rec URLOwnershipRecord
		if err := rows.Scan(
			&rec.ShortCode,
			&rec.Domain,
			&rec.OwnerSub,
			&rec.CreatedAt,
			&rec.DeletedAt,
			&rec.DeletedBy,
		); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

// GetShortCodeSet возвращает set активных short_code владельца.
func (r *URLOwnershipRepository) GetShortCodeSet(ctx context.Context, ownerSub string) (map[string]struct{}, error) {
	rows, err := r.pool.Query(ctx,
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
