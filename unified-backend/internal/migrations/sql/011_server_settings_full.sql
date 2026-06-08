-- +migrate Up
-- server_settings already has key-value schema (005_server_settings.sql).
-- New keys: role_source, admin_role, user_tag_internal_id,
-- cors_allowed_origins, shlink_runner_mode, shlink_container_name,
-- max_visits_default, link_ttl_default_days
-- They are inserted on first startup via SeedFromEnv (ON CONFLICT DO NOTHING).

-- +migrate Down
-- no-op
