-- 009_deactivation_permissions.sql
-- Гранулярные права для деактивации/реактивации и permanent delete.
-- canDeleteOwnLinks/canDeleteAllLinks теперь означают именно soft-delete (деактивацию).
-- Для physical delete — отдельные can_delete_*_permanently.

ALTER TABLE role_permissions
    ADD COLUMN IF NOT EXISTS can_deactivate_own_links         BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS can_deactivate_all_links         BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS can_reactivate_own_links         BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS can_reactivate_all_links         BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS can_delete_own_links_permanently BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS can_delete_all_links_permanently BOOLEAN NOT NULL DEFAULT false;

UPDATE role_permissions SET
    can_deactivate_own_links         = true,
    can_deactivate_all_links         = true,
    can_reactivate_own_links         = true,
    can_reactivate_all_links         = true,
    can_delete_own_links_permanently = true,
    can_delete_all_links_permanently = true
WHERE role = 'admin';

UPDATE role_permissions SET
    can_deactivate_own_links         = true,
    can_deactivate_all_links         = false,
    can_reactivate_own_links         = true,
    can_reactivate_all_links         = false,
    can_delete_own_links_permanently = false,
    can_delete_all_links_permanently = false
WHERE role = 'user';
