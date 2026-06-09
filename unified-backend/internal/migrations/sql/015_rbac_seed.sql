-- +goose Up
-- 015_rbac_seed.sql: seed initial permissions, roles, role_permissions.
-- Существующие пользователи с role='admin' получают роль admin.

-- -------------------------------------------------------
-- 1. Permissions
-- -------------------------------------------------------
INSERT INTO permissions (id, name, description) VALUES
    (gen_random_uuid(), 'dashboard.view',      'Просмотр дашборда'),
    (gen_random_uuid(), 'short_urls.create',   'Создание коротких ссылок'),
    (gen_random_uuid(), 'short_urls.update',   'Редактирование ссылок'),
    (gen_random_uuid(), 'short_urls.delete',   'Удаление ссылок'),
    (gen_random_uuid(), 'short_urls.view_all', 'Просмотр всех ссылок (чужих)'),
    (gen_random_uuid(), 'users.view',          'Просмотр списка пользователей'),
    (gen_random_uuid(), 'users.manage',        'Создание/редактирование/удаление пользователей'),
    (gen_random_uuid(), 'roles.view',          'Просмотр ролей'),
    (gen_random_uuid(), 'roles.manage',        'Управление ролями и разрешениями'),
    (gen_random_uuid(), 'system.config',       'Изменение конфигурации системы')
ON CONFLICT (name) DO NOTHING;

-- -------------------------------------------------------
-- 2. Roles
-- -------------------------------------------------------
INSERT INTO roles (id, name, description) VALUES
    (gen_random_uuid(), 'admin',        'Полный доступ ко всем функциям'),
    (gen_random_uuid(), 'viewer',       'Базовые права: просмотр дашборда, создание/редактирование/удаление своих ссылок'),
    (gen_random_uuid(), 'auditor_admin','Просмотр пользователей/ролей/ссылок, без права создавать/изменять')
ON CONFLICT (name) DO NOTHING;

-- -------------------------------------------------------
-- 3. role_permissions_v2: admin -> all
-- -------------------------------------------------------
INSERT INTO role_permissions_v2 (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'admin'
ON CONFLICT DO NOTHING;

-- -------------------------------------------------------
-- 4. role_permissions_v2: viewer
--    dashboard.view, short_urls.create, short_urls.update, short_urls.delete
-- -------------------------------------------------------
INSERT INTO role_permissions_v2 (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.name IN (
    'dashboard.view',
    'short_urls.create',
    'short_urls.update',
    'short_urls.delete'
)
WHERE r.name = 'viewer'
ON CONFLICT DO NOTHING;

-- -------------------------------------------------------
-- 5. role_permissions_v2: auditor_admin
--    users.view, roles.view, dashboard.view, short_urls.view_all
-- -------------------------------------------------------
INSERT INTO role_permissions_v2 (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.name IN (
    'dashboard.view',
    'short_urls.view_all',
    'users.view',
    'roles.view'
)
WHERE r.name = 'auditor_admin'
ON CONFLICT DO NOTHING;

-- -------------------------------------------------------
-- 6. Миграция существующих пользователей
--    Все существующие пользователи с role='admin' получают role_id=admin.
--    Остальные -> role_id=viewer.
-- -------------------------------------------------------
UPDATE users
SET role_id = (
    SELECT id FROM roles WHERE name = 'admin' LIMIT 1
)
WHERE role = 'admin'
  AND role_id IS NULL;

UPDATE users
SET role_id = (
    SELECT id FROM roles WHERE name = 'viewer' LIMIT 1
)
WHERE role != 'admin'
  AND role_id IS NULL;

-- +goose Down
DELETE FROM role_permissions_v2
WHERE role_id IN (SELECT id FROM roles WHERE name IN ('admin', 'viewer', 'auditor_admin'));

DELETE FROM roles WHERE name IN ('admin', 'viewer', 'auditor_admin');

DELETE FROM permissions WHERE name IN (
    'dashboard.view', 'short_urls.create', 'short_urls.update',
    'short_urls.delete', 'short_urls.view_all',
    'users.view', 'users.manage', 'roles.view', 'roles.manage', 'system.config'
);
