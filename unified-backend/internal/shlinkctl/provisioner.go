package shlinkctl

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Provisioner выдаёт shlink API-ключи пользователям.
// Потокобезопасен: использует advisory lock PostgreSQL на время генерации.
type Provisioner struct {
	pool   *pgxpool.Pool
	runner Runner
}

func NewProvisioner(pool *pgxpool.Pool, runner Runner) *Provisioner {
	return &Provisioner{pool: pool, runner: runner}
}

// EnsureAPIKey — идемпотентно выдаёт ключ пользователю.
//
// Алгоритм:
//  1. Быстрая проверка: если shlink_api_key != '' → возвращаем существующий ключ.
//  2. Берём pg_advisory_lock по FNV-1a hash(sub).
//  3. Re-check после блокировки (double-checked locking).
//  4. Удаляем старый ключ в shlink по имени (если остался от предыдущего запуска).
//  5. Генерируем новый ключ через CLI.
//  6. Записываем ключ в БД.
//  7. Освобождаем lock.
func (p *Provisioner) EnsureAPIKey(ctx context.Context, sub, username string) (string, error) {
	// Шаг 1: быстрая проверка без лока
	existingKey, err := p.getAPIKey(ctx, sub)
	if err != nil {
		return "", fmt.Errorf("provisioner: get key: %w", err)
	}
	if existingKey != "" {
		return existingKey, nil
	}

	// Шаг 2: advisory lock
	lockID := hashSub(sub)

	conn, err := p.pool.Acquire(ctx)
	if err != nil {
		return "", fmt.Errorf("provisioner: acquire conn: %w", err)
	}
	defer conn.Release()

	if _, err = conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, lockID); err != nil {
		return "", fmt.Errorf("provisioner: advisory lock: %w", err)
	}
	defer func() {
		if _, unlockErr := conn.Exec(ctx, `SELECT pg_advisory_unlock($1)`, lockID); unlockErr != nil {
			slog.Warn("provisioner: advisory unlock failed", "sub", sub, "err", unlockErr)
		}
	}()

	// Шаг 3: re-check после лока
	var recheck string
	if err = conn.QueryRow(ctx, `SELECT COALESCE(shlink_api_key, '') FROM users WHERE sub = $1`, sub).Scan(&recheck); err != nil {
		return "", fmt.Errorf("provisioner: re-check key: %w", err)
	}
	if recheck != "" {
		return recheck, nil
	}

	// Шаг 4: удалить старый ключ по имени (idempotent — не падает если нет)
	slog.Info("provisioner: deleting stale api key if exists", "sub", sub, "username", username)
	if delErr := p.runner.DeleteAPIKey(ctx, username); delErr != nil {
		// Не фатально — логируем и продолжаем
		slog.Warn("provisioner: delete stale key failed", "sub", sub, "username", username, "err", delErr)
	}

	// Шаг 5: генерация через CLI
	slog.Info("provisioner: generating api key", "sub", sub, "username", username)
	newKey, err := p.runner.GenerateAPIKey(ctx, username)
	if err != nil {
		return "", fmt.Errorf("provisioner: generate key for %q: %w", username, err)
	}

	// Шаг 6: запись в БД
	if _, err = conn.Exec(ctx, `UPDATE users SET shlink_api_key = $1, updated_at = NOW() WHERE sub = $2`, newKey, sub); err != nil {
		return "", fmt.Errorf("provisioner: store key: %w", err)
	}

	slog.Info("provisioner: api key stored", "sub", sub, "username", username)
	return newKey, nil
}

func (p *Provisioner) getAPIKey(ctx context.Context, sub string) (string, error) {
	var key string
	if err := p.pool.QueryRow(ctx, `SELECT COALESCE(shlink_api_key, '') FROM users WHERE sub = $1`, sub).Scan(&key); err != nil {
		return "", err
	}
	return key, nil
}

// hashSub возвращает int64 advisory lock ID для sub.
func hashSub(sub string) int64 {
	const (
		offset64 = uint64(14695981039346656037)
		prime64  = uint64(1099511628211)
	)
	h := offset64
	for i := 0; i < len(sub); i++ {
		h ^= uint64(sub[i])
		h *= prime64
	}
	return int64(h)
}
