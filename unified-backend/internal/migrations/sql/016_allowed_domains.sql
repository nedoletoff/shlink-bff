-- 016_allowed_domains.sql
-- Добавляет колонку allowed_domains в таблицу users.
-- Хранит JSON-массив разрешённых доменов для создания коротких ссылок.
-- Пустая строка или '[]' = ограничений нет.
-- Идемпотентно через IF NOT EXISTS.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS allowed_domains TEXT NOT NULL DEFAULT '';

COMMENT ON COLUMN users.allowed_domains IS
    'JSON array of allowed domains for short URL creation, e.g. ["example.com","short.io"]. Empty string means no restriction.';
