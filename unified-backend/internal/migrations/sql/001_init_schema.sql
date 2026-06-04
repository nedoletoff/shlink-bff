-- migrations/001_init_schema.sql
-- Инициализация схемы БД для unified-backend

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Пользователи: основная таблица идентификаторов
CREATE TABLE IF NOT EXISTS users (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    sub             TEXT        NOT NULL UNIQUE,
    username        TEXT        NOT NULL,
    email           TEXT        NOT NULL,
    role            TEXT        NOT NULL DEFAULT 'user'
                                CHECK (role IN ('admin', 'user')),
    shlink_api_key  TEXT        NOT NULL DEFAULT '',
    slug_prefix     TEXT,
    status          TEXT        NOT NULL DEFAULT 'active'
                                CHECK (status IN ('active', 'disabled', 'pending')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_sub    ON users (sub);
CREATE INDEX        IF NOT EXISTS idx_users_role   ON users (role);
CREATE INDEX        IF NOT EXISTS idx_users_status ON users (status);

-- Теги пользователей: изоляция тегов (feature: userTagInternalIdEnabled)
CREATE TABLE IF NOT EXISTS user_tags (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tag_name    TEXT        NOT NULL,
    internal_id TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, tag_name)
);

CREATE INDEX IF NOT EXISTS idx_user_tags_user_id     ON user_tags (user_id);
CREATE INDEX IF NOT EXISTS idx_user_tags_internal_id ON user_tags (internal_id);

-- Журнал аудита
CREATE TABLE IF NOT EXISTS audit_logs (
    id          BIGSERIAL   PRIMARY KEY,
    user_sub    TEXT        NOT NULL,
    username    TEXT,
    role        TEXT,
    action      TEXT        NOT NULL,
    resource    TEXT,
    result      TEXT        NOT NULL
                            CHECK (result IN ('success', 'denied', 'error')),
    details     JSONB,
    ip_address  TEXT,
    user_agent  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_user_sub   ON audit_logs (user_sub);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action     ON audit_logs (action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_result     ON audit_logs (result);
CREATE INDEX IF NOT EXISTS idx_audit_logs_sub_ts     ON audit_logs (user_sub, created_at DESC);
