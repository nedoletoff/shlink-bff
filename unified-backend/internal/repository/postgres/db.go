package postgres

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	// Настройка пула (#17): по умолчанию pgxpool даёт MaxConns=4, что под нагрузкой
	// мгновенно исчерпывается. DSN-параметры (pool_max_conns и т.п.) имеют приоритет:
	// здесь задаём разумные дефолты, если они не переопределены.
	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	slog.Info("postgres: connected successfully",
		"max_conns", cfg.MaxConns, "min_conns", cfg.MinConns)
	return pool, nil
}
