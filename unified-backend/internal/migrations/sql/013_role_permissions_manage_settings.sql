-- migrations/013_role_permissions_manage_settings.sql
-- Добавляет разрешение can_manage_settings в role_permissions.

ALTER TABLE role_permissions
    ADD COLUMN IF NOT EXISTS can_manage_settings BOOLEAN NOT NULL DEFAULT false;

-- Администратору даём can_manage_settings сразу.
UPDATE role_permissions
SET can_manage_settings = true
WHERE role = 'admin';
