-- +migrate Up
ALTER TABLE url_ownership
    ADD COLUMN IF NOT EXISTS owner_username TEXT NOT NULL DEFAULT '';

-- +migrate Down
ALTER TABLE url_ownership
    DROP COLUMN IF EXISTS owner_username;
