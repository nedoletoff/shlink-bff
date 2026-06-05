package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ServerSettingsRepository хранит key-value настройки сервера в БД.
// Значения переопределяют env-конфиг при config_source=db.
type ServerSettingsRepository struct {
	pool *pgxpool.Pool
}

func NewServerSettingsRepository(pool *pgxpool.Pool) *ServerSettingsRepository {
	return &ServerSettingsRepository{pool: pool}
}

// Get возвращает значение по ключу. Если ключ не найден — возвращает ("", nil).
func (r *ServerSettingsRepository) Get(ctx context.Context, key string) (string, error) {
	var value string
	err := r.pool.QueryRow(ctx,
		`SELECT value FROM server_settings WHERE key = $1`,
		key,
	).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return value, err
}

// GetAll возвращает все настройки в виде map[key]value.
func (r *ServerSettingsRepository) GetAll(ctx context.Context) (map[string]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT key, value FROM server_settings ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		result[k] = v
	}
	return result, rows.Err()
}

// Set сохраняет или обновляет значение ключа.
func (r *ServerSettingsRepository) Set(ctx context.Context, key, value, updatedBy string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO server_settings (key, value, updated_at, updated_by)
		 VALUES ($1, $2, now(), $3)
		 ON CONFLICT (key) DO UPDATE
		   SET value = EXCLUDED.value,
		       updated_at = now(),
		       updated_by = EXCLUDED.updated_by`,
		key, value, updatedBy,
	)
	return err
}

// SetMany сохраняет несколько ключей в одной транзакции.
func (r *ServerSettingsRepository) SetMany(ctx context.Context, kv map[string]string, updatedBy string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	for k, v := range kv {
		_, err := tx.Exec(ctx,
			`INSERT INTO server_settings (key, value, updated_at, updated_by)
			 VALUES ($1, $2, now(), $3)
			 ON CONFLICT (key) DO UPDATE
			   SET value = EXCLUDED.value,
			       updated_at = now(),
			       updated_by = EXCLUDED.updated_by`,
			k, v, updatedBy,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
