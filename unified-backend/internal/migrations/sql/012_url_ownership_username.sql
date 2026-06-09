-- 012_url_ownership_username.sql
-- Добавляем колонку owner_username, которую ожидает url_ownership_repo.go.
-- Таблица создана в 004, но поле не было добавлено ни в одной из последующих миграций.
-- DEFAULT '' — чтобы не ломать существующие строки.

ALTER TABLE url_ownership
    ADD COLUMN IF NOT EXISTS owner_username TEXT NOT NULL DEFAULT '';
