-- 006_url_ownership_active.sql
-- Add is_active flag to url_ownership.
-- false = deactivated (link exists in Shlink with enabled=false, history preserved)
-- true  = active (default)

ALTER TABLE url_ownership
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE;

CREATE INDEX IF NOT EXISTS url_ownership_is_active_idx
    ON url_ownership (owner_sub, is_active)
    WHERE deleted_at IS NULL;
