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
	IsActive  bool
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

// Save сохраняет новую запись ownership (is_active=true).
// При конфликте — обновляет is_active=true и сбрасывает deleted_at (повторное создание).
func (r *URLOwnershipRepository) Save(ctx context.Context, shortCode, ownerSub, domain string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO url_ownership (short_code, domain, owner_sub, is_active)
		 VALUES ($1, $2, $3, TRUE)
		 ON CONFLICT (short_code, domain)
		 DO UPDATE SET owner_sub = EXCLUDED.owner_sub,
		               is_active = TRUE,
		               deleted_at = NULL,
		               deleted_by = NULL`,
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

// HardDelete удаляет запись из url_ownership полностью (физическое удаление).
// Вызывается вместе с DELETE в Shlink — освобождает slug для повторного использования.
func (r *URLOwnershipRepository) HardDelete(ctx context.Context, shortCode, domain string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM url_ownership WHERE short_code = $1 AND domain = $2`,
		shortCode, domain,
	)
	return err
}

// SetActive устанавливает is_active для ссылки (деактивация/активация).
func (r *URLOwnershipRepository) SetActive(ctx context.Context, shortCode, domain string, active bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE url_ownership
		    SET is_active = $3
		  WHERE short_code = $1 AND domain = $2
		    AND deleted_at IS NULL`,
		shortCode, domain, active,
	)
	return err
}

// GetByOwner возвращает все активные (не удалённые) ссылки пользователя.
func (r *URLOwnershipRepository) GetByOwner(ctx context.Context, ownerSub string) ([]URLOwnershipRecord, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT short_code, domain, owner_sub, is_active, created_at, deleted_at, deleted_by
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
			&rec.IsActive,
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

// GetShortCodeSet возвращает set всех (активных и деактивированных) short_code владельца,
// которые не удалены. Используется для фильтрации списка ссылок.
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

// GetActiveCodeSet возвращает set только активных (is_active=true, не удалённых) short_code владельца.
// Используется для подсчёта «живых» ссылок на дашборде.
func (r *URLOwnershipRepository) GetActiveCodeSet(ctx context.Context, ownerSub string) (map[string]struct{}, error) {
	rows, err := r.pool.Query(ctx,
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
