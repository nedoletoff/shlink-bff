-- Новая схема: единая RBAC-модель, расширенная таблица url_ownership, аудит, настройки

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- Таблицы разрешений и ролей (новая модель)
-- ============================================================

CREATE TABLE IF NOT EXISTS permissions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT UNIQUE NOT NULL,
    description TEXT
);

CREATE TABLE IF NOT EXISTS roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT UNIQUE NOT NULL,
    description TEXT
);

CREATE TABLE IF NOT EXISTS role_permissions_v2 (
    role_id       UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- ============================================================
-- Пользователи (роль через role_id, денормализованное поле role_text)
-- ============================================================

CREATE TABLE IF NOT EXISTS users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sub             TEXT NOT NULL UNIQUE,
    username        TEXT NOT NULL,
    email           TEXT NOT NULL,
    role_id         UUID REFERENCES roles(id),
    role_text       TEXT NOT NULL DEFAULT '',
    shlink_api_key  TEXT NOT NULL DEFAULT '',
    slug_prefix     TEXT,
    allowed_domains TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'active'
                        CHECK (status IN ('active', 'disabled', 'pending')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE OR REPLACE FUNCTION sync_role_text() RETURNS TRIGGER AS $$
BEGIN
    IF NEW.role_id IS NOT NULL THEN
        SELECT name INTO NEW.role_text FROM roles WHERE id = NEW.role_id;
    ELSE
        NEW.role_text := '';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_sync_role_text ON users;
CREATE TRIGGER trigger_sync_role_text
    BEFORE INSERT OR UPDATE OF role_id ON users
    FOR EACH ROW EXECUTE FUNCTION sync_role_text();

CREATE INDEX IF NOT EXISTS idx_users_sub        ON users (sub);
CREATE INDEX IF NOT EXISTS idx_users_role_id    ON users (role_id);
CREATE INDEX IF NOT EXISTS idx_users_status     ON users (status);

-- ============================================================
-- Теги пользователей (изоляция тегов)
-- ============================================================

CREATE TABLE IF NOT EXISTS user_tags (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tag_name    TEXT NOT NULL,
    internal_id TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, tag_name)
);
CREATE INDEX IF NOT EXISTS idx_user_tags_user_id     ON user_tags (user_id);
CREATE INDEX IF NOT EXISTS idx_user_tags_internal_id ON user_tags (internal_id);

-- ============================================================
-- Владение короткими ссылками (с расширенными полями)
-- ============================================================

CREATE TABLE IF NOT EXISTS url_ownership (
    short_code     TEXT NOT NULL,
    domain         TEXT NOT NULL DEFAULT '',
    owner_sub      TEXT NOT NULL,
    owner_username TEXT NOT NULL DEFAULT '',
    title          TEXT,
    is_active      BOOLEAN NOT NULL DEFAULT TRUE,
    valid_since    TIMESTAMPTZ,
    valid_until    TIMESTAMPTZ,
    max_visits     INT,
    is_public      BOOLEAN NOT NULL DEFAULT FALSE,
    tags           TEXT[],
    deactivated_at TIMESTAMPTZ,
    deactivated_by TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at     TIMESTAMPTZ,
    deleted_by     TEXT,
    PRIMARY KEY (short_code, domain)
);

CREATE INDEX IF NOT EXISTS idx_url_ownership_owner_sub    ON url_ownership (owner_sub) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_url_ownership_is_active    ON url_ownership (owner_sub, is_active) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_url_ownership_deleted_at   ON url_ownership (deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_url_ownership_valid_until  ON url_ownership (valid_until) WHERE valid_until IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_url_ownership_is_public    ON url_ownership (is_public) WHERE is_public = true;

-- ============================================================
-- Системные настройки
-- ============================================================

CREATE TABLE IF NOT EXISTS server_settings (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by TEXT
);
INSERT INTO server_settings (key, value, updated_by)
    VALUES ('config_source', 'env', 'system')
    ON CONFLICT (key) DO NOTHING;

-- ============================================================
-- Аудит
-- ============================================================

CREATE TABLE IF NOT EXISTS audit_logs (
    id         BIGSERIAL PRIMARY KEY,
    user_sub   TEXT NOT NULL,
    username   TEXT,
    action     TEXT NOT NULL,
    resource   TEXT,
    result     TEXT NOT NULL CHECK (result IN ('success', 'denied', 'error')),
    details    JSONB,
    ip_address TEXT,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_user_sub   ON audit_logs (user_sub);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action     ON audit_logs (action);
