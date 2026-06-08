-- 008_url_deactivation.sql
-- Добавляем audit-trail поля для деактивации ссылок.
-- is_active остаётся как денормализованный флаг (быстрые запросы),
-- deactivated_at/by — для audit trail и карточки ссылки.

ALTER TABLE url_ownership
    ADD COLUMN IF NOT EXISTS deactivated_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS deactivated_by TEXT;

-- Индекс для фильтра status=inactive
CREATE INDEX IF NOT EXISTS url_ownership_deactivated_at_owner_idx
    ON url_ownership (owner_sub, deactivated_at)
    WHERE deleted_at IS NULL;
