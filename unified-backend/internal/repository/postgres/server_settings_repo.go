package postgres

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"unified-backend/internal/config"
)

// ServerSettingsRepository хранит key-value настройки сервера в БД.
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

// SeedFromEnv записывает все поля из cfg в БД только если ключ ещё не существует.
// Идемпотентно: ON CONFLICT DO NOTHING.
func (r *ServerSettingsRepository) SeedFromEnv(ctx context.Context, cfg *config.Config) error {
	seeds := map[string]string{
		"short_code_length":     strconv.Itoa(cfg.ShlinkShortIDLength),
		"allow_custom_slugs":    strconv.FormatBool(cfg.UserCustomSlugEnabled),
		"user_slug_prefix":      strconv.FormatBool(cfg.UserSlugPrefixEnabled),
		"default_domain":        cfg.ShlinkDefaultDomain,
		"role_source":           string(cfg.RoleSource),
		"admin_role":            cfg.AdminRole,
		"user_tag_internal_id":  strconv.FormatBool(cfg.UserTagInternalIdEnabled),
		"cors_allowed_origins":  strings.Join(cfg.CORSAllowedOrigins, ","),
		"shlink_runner_mode":    cfg.ShlinkRunnerMode,
		"shlink_container_name": cfg.ShlinkContainerName,
		"max_visits_default":    strconv.Itoa(cfg.MaxVisitsDefault),
		"link_ttl_default_days": strconv.Itoa(cfg.LinkTtlDefaultDays),
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	for k, v := range seeds {
		_, err := tx.Exec(ctx,
			`INSERT INTO server_settings (key, value, updated_at, updated_by)
			 VALUES ($1, $2, now(), 'system')
			 ON CONFLICT (key) DO NOTHING`,
			k, v,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// ApplyAll читает все ключи из БД и применяет к cfg.
// Вызывается после SeedFromEnv — переопределяет env значения данными из БД.
func (r *ServerSettingsRepository) ApplyAll(ctx context.Context, cfg *config.Config) error {
	settings, err := r.GetAll(ctx)
	if err != nil {
		return err
	}

	if v, ok := settings["short_code_length"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.ShlinkShortIDLength = n
		}
	}
	if v, ok := settings["allow_custom_slugs"]; ok {
		cfg.UserCustomSlugEnabled = v == "true"
	}
	if v, ok := settings["user_slug_prefix"]; ok {
		cfg.UserSlugPrefixEnabled = v == "true"
	}
	if v, ok := settings["default_domain"]; ok && v != "" {
		cfg.ShlinkDefaultDomain = v
	}
	if v, ok := settings["role_source"]; ok && v != "" {
		cfg.RoleSource = config.RoleSource(v)
	}
	if v, ok := settings["admin_role"]; ok && v != "" {
		cfg.AdminRole = v
	}
	if v, ok := settings["user_tag_internal_id"]; ok {
		cfg.UserTagInternalIdEnabled = v == "true"
	}
	if v, ok := settings["cors_allowed_origins"]; ok && v != "" {
		var origins []string
		for _, o := range strings.Split(v, ",") {
			if s := strings.TrimSpace(o); s != "" {
				origins = append(origins, s)
			}
		}
		if len(origins) > 0 {
			cfg.CORSAllowedOrigins = origins
		}
	}
	if v, ok := settings["shlink_runner_mode"]; ok && v != "" {
		cfg.ShlinkRunnerMode = v
	}
	if v, ok := settings["shlink_container_name"]; ok && v != "" {
		cfg.ShlinkContainerName = v
	}
	if v, ok := settings["max_visits_default"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.MaxVisitsDefault = n
		}
	}
	if v, ok := settings["link_ttl_default_days"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.LinkTtlDefaultDays = n
		}
	}
	return nil
}
