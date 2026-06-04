-- migrations/003_role_permissions.sql
-- Гранулярные разрешения на уровне роли.
-- Администратор сервиса настраивает через /api/admin/roles/{role}/permissions.

CREATE TABLE IF NOT EXISTS role_permissions (
    role                        TEXT    PRIMARY KEY,

    -- Ссылки: просмотр
    can_view_own_links          BOOLEAN NOT NULL DEFAULT true,
    can_view_all_links          BOOLEAN NOT NULL DEFAULT false,

    -- Ссылки: создание
    can_create_links            BOOLEAN NOT NULL DEFAULT true,
    can_create_with_custom_slug BOOLEAN NOT NULL DEFAULT false,  -- явный customSlug
    can_create_without_slug     BOOLEAN NOT NULL DEFAULT true,   -- slug генерирует Shlink

    -- Ссылки: редактирование
    can_edit_own_links          BOOLEAN NOT NULL DEFAULT true,
    can_edit_all_links          BOOLEAN NOT NULL DEFAULT false,

    -- Ссылки: удаление
    can_delete_own_links        BOOLEAN NOT NULL DEFAULT true,
    can_delete_all_links        BOOLEAN NOT NULL DEFAULT false,

    -- Теги
    can_manage_own_tags         BOOLEAN NOT NULL DEFAULT true,
    can_manage_all_tags         BOOLEAN NOT NULL DEFAULT false,

    -- Статистика
    can_view_own_stats          BOOLEAN NOT NULL DEFAULT true,
    can_view_all_stats          BOOLEAN NOT NULL DEFAULT false,

    -- Аудит и управление
    can_view_audit_logs         BOOLEAN NOT NULL DEFAULT false,
    can_manage_users            BOOLEAN NOT NULL DEFAULT false,
    can_manage_roles            BOOLEAN NOT NULL DEFAULT false,

    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Дефолтные роли при первом запуске.
-- Значения можно изменить через /api/admin/roles/{role}/permissions.
INSERT INTO role_permissions (role,
    can_view_own_links, can_view_all_links,
    can_create_links, can_create_with_custom_slug, can_create_without_slug,
    can_edit_own_links, can_edit_all_links,
    can_delete_own_links, can_delete_all_links,
    can_manage_own_tags, can_manage_all_tags,
    can_view_own_stats, can_view_all_stats,
    can_view_audit_logs, can_manage_users, can_manage_roles
) VALUES
-- admin: полные права
('admin',
    true, true,
    true, true, true,
    true, true,
    true, true,
    true, true,
    true, true,
    true, true, true
),
-- user: только свои ссылки, без custom slug
('user',
    true,  false,
    true,  false, true,
    true,  false,
    true,  false,
    true,  false,
    true,  false,
    false, false, false
)
ON CONFLICT (role) DO NOTHING;
