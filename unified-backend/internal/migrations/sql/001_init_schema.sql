-- migrations/001_init_schema.sql
-- Полная начальная схема БД для unified-backend (RBAC-модель).

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Атомарные действия, которые можно проверять
CREATE TABLE IF NOT EXISTS permissions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT UNIQUE NOT NULL,  -- 'dashboard.view', 'users.manage', ...
    description TEXT
);

-- Роли пользователей
CREATE TABLE IF NOT EXISTS roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT UNIQUE NOT NULL,  -- 'admin', 'viewer', 'auditor_admin', ...
    description TEXT
);

-- Связь ролей с разрешениями (many-to-many)
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id       UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- Пользователи
CREATE TABLE IF NOT EXISTS users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    sub           TEXT        NOT NULL UNIQUE,
    username      TEXT        NOT NULL,
    email         TEXT        NOT NULL,
    role_id       UUID        REFERENCES roles(id),
    shlink_api_key TEXT       NOT NULL DEFAULT '',
    slug_prefix   TEXT,
    status        TEXT        NOT NULL DEFAULT 'active'
                              CHECK (status IN ('active', 'disabled', 'pending')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_sub     ON users (sub);
CREATE INDEX        IF NOT EXISTS idx_users_role_id  ON users (role_id);
CREATE INDEX        IF NOT EXISTS idx_users_status   ON users (status);
CREATE INDEX        IF NOT EXISTS idx_rp_role_id     ON role_permissions (role_id);
CREATE INDEX        IF NOT EXISTS idx_rp_perm_id     ON role_permissions (permission_id);

-- Теги пользователей (изоляция тегов)
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

-- Ownership коротких ссылок
CREATE TABLE IF NOT EXISTS url_ownership (
    short_code     TEXT        NOT NULL,
    domain         TEXT        NOT NULL DEFAULT '',
    owner_sub      TEXT        NOT NULL,
    owner_username TEXT        NOT NULL DEFAULT '',
    is_active      BOOLEAN     NOT NULL DEFAULT TRUE,
    deactivated_at TIMESTAMPTZ,
    deactivated_by TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at     TIMESTAMPTZ,
    deleted_by     TEXT,
    PRIMARY KEY (short_code, domain)
);

CREATE INDEX IF NOT EXISTS idx_url_ownership_owner_sub
    ON url_ownership (owner_sub)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_url_ownership_is_active
    ON url_ownership (owner_sub, is_active)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_url_ownership_deleted_at
    ON url_ownership (deleted_at)
    WHERE deleted_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_url_ownership_deactivated_at
    ON url_ownership (owner_sub, deactivated_at)
    WHERE deleted_at IS NULL;

-- Системные настройки (key-value)
CREATE TABLE IF NOT EXISTS server_settings (
    key        TEXT        PRIMARY KEY,
    value      TEXT        NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by TEXT
);

INSERT INTO server_settings (key, value, updated_by)
VALUES ('config_source', 'env', 'system')
ON CONFLICT (key) DO NOTHING;

-- Журнал аудита
CREATE TABLE IF NOT EXISTS audit_logs (
    id         BIGSERIAL   PRIMARY KEY,
    user_sub   TEXT        NOT NULL,
    username   TEXT,
    action     TEXT        NOT NULL,
    resource   TEXT,
    result     TEXT        NOT NULL
                           CHECK (result IN ('success', 'denied', 'error')),
    details    JSONB,
    ip_address TEXT,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_user_sub   ON audit_logs (user_sub);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action     ON audit_logs (action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_result     ON audit_logs (result);
CREATE INDEX IF NOT EXISTS idx_audit_logs_sub_ts     ON audit_logs (user_sub, created_at DESC);
