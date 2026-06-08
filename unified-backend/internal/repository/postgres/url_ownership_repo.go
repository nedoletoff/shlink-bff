package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// URLOwnershipRecord — запись в url_ownership.
type URLOwnershipRecord struct {
	ShortCode     string
	Domain        string
	OwnerSub      string
	OwnerUsername string
	IsActive      bool
	DeactivatedAt *time.Time
	DeactivatedBy *string
	CreatedAt     time.Time
	DeletedAt     *time.Time
	DeletedBy     *string
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
func (r *URLOwnershipRepository) Save(ctx context.Context, shortCode, ownerSub, ownerUsername, domain string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO url_ownership (short_code, domain, owner_sub, owner_username, is_active)
		 VALUES ($1, $2, $3, $4, TRUE)
		 ON CONFLICT (short_code, domain)
		 DO UPDATE SET owner_sub      = EXCLUDED.owner_sub,
		               owner_username = EXCLUDED.owner_username,
		               is_active      = TRUE,
		               deactivated_at = NULL,
		               deactivated_by = NULL,
		               deleted_at     = NULL,
		               deleted_by     = NULL`,
		shortCode, domain, ownerSub, ownerUsername,
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

// GetOwnership возвращает полную запись url_ownership для short_code+domain.
func (r *URLOwnershipRepository) GetOwnership(ctx context.Context, shortCode, domain string) (*URLOwnershipRecord, error) {
	var rec URLOwnershipRecord
	err := r.pool.QueryRow(ctx,
		`SELECT short_code, domain, owner_sub, owner_username, is_active,
		        deactivated_at, deactivated_by,
		        created_at, deleted_at, deleted_by
		   FROM url_ownership
		  WHERE short_code = $1 AND domain = $2`,
		shortCode, domain,
	).Scan(
		&rec.ShortCode, &rec.Domain, &rec.OwnerSub, &rec.OwnerUsername, &rec.IsActive,
		&rec.DeactivatedAt, &rec.DeactivatedBy,
		&rec.CreatedAt, &rec.DeletedAt, &rec.DeletedBy,
	)
	if err != nil {
		return nil, err
	}
	return &rec, nil
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

// Deactivate устанавливает is_active=false + заполняет deactivated_at/by.
// Идемпотентно: повторная деактивация не меняет deactivated_at (первый timestamp сохраняется).
func (r *URLOwnershipRepository) Deactivate(ctx context.Context, shortCode, domain, actorSub string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE url_ownership
		    SET is_active      = FALSE,
		        deactivated_at = COALESCE(deactivated_at, NOW()),
		        deactivated_by = COALESCE(deactivated_by, $3)
		  WHERE short_code = $1 AND domain = $2
		    AND deleted_at IS NULL`,
		shortCode, domain, actorSub,
	)
	return err
}

// Activate сбрасывает деактивацию.
// Идемпотентно: повторная активация уже активной ссылки — no-op.
func (r *URLOwnershipRepository) Activate(ctx context.Context, shortCode, domain string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE url_ownership
		    SET is_active      = TRUE,
		        deactivated_at = NULL,
		        deactivated_by = NULL
		  WHERE short_code = $1 AND domain = $2
		    AND deleted_at IS NULL`,
		shortCode, domain,
	)
	return err
}

// SoftDelete помечает запись как удалённую (permanent delete в терминах BFF).
// Физически строка остаётся — для аудита. slug освобождается логически.
func (r *URLOwnershipRepository) SoftDelete(ctx context.Context, shortCode, domain, actorSub string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE url_ownership
		    SET deleted_at = NOW(),
		        deleted_by = $3,
		        is_active  = FALSE
		  WHERE short_code = $1 AND domain = $2
		    AND deleted_at IS NULL`,
		shortCode, domain, actorSub,
	)
	return err
}

// HardDelete удаляет запись из url_ownership полностью (физическое удаление).
// Оставлен для обратной совместимости; в новом коде используйте SoftDelete.
func (r *URLOwnershipRepository) HardDelete(ctx context.Context, shortCode, domain string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM url_ownership WHERE short_code = $1 AND domain = $2`,
		shortCode, domain,
	)
	return err
}

// SetActive устанавливает is_active для ссылки (legacy toggle).
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
		`SELECT short_code, domain, owner_sub, owner_username, is_active,
		        deactivated_at, deactivated_by,
		        created_at, deleted_at, deleted_by
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
			&rec.ShortCode, &rec.Domain, &rec.OwnerSub, &rec.OwnerUsername, &rec.IsActive,
			&rec.DeactivatedAt, &rec.DeactivatedBy,
			&rec.CreatedAt, &rec.DeletedAt, &rec.DeletedBy,
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

// GetStatusCodeSet возвращает map[shortCode]isActive для владельца (не удалённых ссылок).
// Используется для обогащения списка ссылок статусом деактивации.
func (r *URLOwnershipRepository) GetStatusCodeSet(ctx context.Context, ownerSub string) (map[string]bool, error) {
	rows, err := r.pool.Query(ctx,
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
