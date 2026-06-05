-- 004_url_ownership.sql
-- Явная модель ownership для коротких ссылок.
-- Используется вместо SlugPrefix-фильтрации.

CREATE TABLE IF NOT EXISTS url_ownership (
    short_code  TEXT        NOT NULL,
    domain      TEXT        NOT NULL DEFAULT '',
    owner_sub   TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ,
    deleted_by  TEXT,
    PRIMARY KEY (short_code, domain)
);

CREATE INDEX IF NOT EXISTS url_ownership_owner_sub_idx
    ON url_ownership (owner_sub)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS url_ownership_deleted_at_idx
    ON url_ownership (deleted_at)
    WHERE deleted_at IS NOT NULL;
