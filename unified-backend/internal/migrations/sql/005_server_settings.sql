-- +migrate Up
CREATE TABLE IF NOT EXISTS server_settings (
    key        TEXT PRIMARY KEY,
    value      TEXT        NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by TEXT
);

-- Флаг источника истины: 'env' — настройки берутся из env-переменных,
-- 'db' — значения из этой таблицы переопределяют env при старте.
INSERT INTO server_settings (key, value, updated_by)
VALUES ('config_source', 'env', 'system')
ON CONFLICT (key) DO NOTHING;
