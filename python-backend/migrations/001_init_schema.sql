-- migrations/001_init_schema.sql
-- Инициализация схемы для python-backend (FastAPI) -- SQLite edition
-- Примечание: при использовании SQLAlchemy + init_db() этот файл не нужен —
-- таблицы создаются через Base.metadata.create_all() при старте приложения.
-- Оставлен для референса и ручного применения.

PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;

-- Пользователи: основная таблица идентификаторов
CREATE TABLE IF NOT EXISTS users (
    id           TEXT        NOT NULL PRIMARY KEY,  -- UUID v4 string
    sub          TEXT        NOT NULL UNIQUE,
    username     TEXT        NOT NULL,
    email        TEXT        NOT NULL,
    display_name TEXT        NOT NULL DEFAULT '',
    role         TEXT        NOT NULL DEFAULT 'user'
                             CHECK (role IN ('admin', 'user')),
    shlink_api_key TEXT      NOT NULL DEFAULT '',
    slug_prefix  TEXT,
    status       TEXT        NOT NULL DEFAULT 'active'
                             CHECK (status IN ('active', 'disabled', 'pending')),
    created_at   TEXT        NOT NULL DEFAULT (datetime('now')),
    updated_at   TEXT        NOT NULL DEFAULT (datetime('now'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_sub    ON users (sub);
CREATE        INDEX IF NOT EXISTS idx_users_role   ON users (role);
CREATE        INDEX IF NOT EXISTS idx_users_status ON users (status);

-- Теги пользователей (фичер userTagInternalIdEnabled)
CREATE TABLE IF NOT EXISTS user_tags (
    id          TEXT  NOT NULL PRIMARY KEY,
    user_id     TEXT  NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    tag_name    TEXT  NOT NULL,
    internal_id TEXT,
    created_at  TEXT  NOT NULL DEFAULT (datetime('now')),
    UNIQUE (user_id, tag_name)
);

CREATE INDEX IF NOT EXISTS idx_user_tags_user_id     ON user_tags (user_id);
CREATE INDEX IF NOT EXISTS idx_user_tags_internal_id ON user_tags (internal_id);

-- Журнал аудита
CREATE TABLE IF NOT EXISTS audit_logs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_sub    TEXT    NOT NULL,
    username    TEXT,
    role        TEXT,
    action      TEXT    NOT NULL,
    resource    TEXT,
    result      TEXT    NOT NULL,
    details     TEXT,  -- JSON хранится как TEXT
    ip_address  TEXT,
    user_agent  TEXT,
    created_at  TEXT    NOT NULL DEFAULT (datetime('now')),
    user_id     TEXT    REFERENCES users (id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_user_sub ON audit_logs (user_sub);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action   ON audit_logs (action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created  ON audit_logs (created_at);
