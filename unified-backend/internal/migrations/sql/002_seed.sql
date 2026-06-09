-- migrations/002_seed.sql
-- Начальные данные: разрешения, роли, назначение разрешений ролям.
-- Миграция для переноса существующих пользователей на роль admin.

-- Разрешения
INSERT INTO permissions (id, name, description) VALUES
    (gen_random_uuid(), 'dashboard.view',     'Просмотр дашборда'),
    (gen_random_uuid(), 'short_urls.create',  'Создание коротких ссылок'),
    (gen_random_uuid(), 'short_urls.update',  'Редактирование ссылок'),
    (gen_random_uuid(), 'short_urls.delete',  'Удаление ссылок'),
    (gen_random_uuid(), 'short_urls.view_all','Просмотр всех ссылок (чужих)'),
    (gen_random_uuid(), 'users.view',         'Просмотр списка пользователей'),
    (gen_random_uuid(), 'users.manage',       'Создание/редактирование/удаление пользователей'),
    (gen_random_uuid(), 'roles.view',         'Просмотр ролей'),
    (gen_random_uuid(), 'roles.manage',       'Управление ролями и их разрешениями'),
    (gen_random_uuid(), 'system.config',      'Изменение конфигурации системы')
ON CONFLICT (name) DO NOTHING;

-- Роли
INSERT INTO roles (id, name, description) VALUES
    (gen_random_uuid(), 'admin',        'Полные права на всё'),
    (gen_random_uuid(), 'viewer',       'Базовый доступ: просмотр и работа со своими ссылками'),
    (gen_random_uuid(), 'auditor_admin','Просмотр пользователей и ролей, без создания ссылок')
ON CONFLICT (name) DO NOTHING;

-- Назначение разрешений ролям
-- admin — все разрешения
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'admin'
ON CONFLICT DO NOTHING;

-- viewer — dashboard.view, short_urls.create, short_urls.update, short_urls.delete
INSERT INTO role_permissions (role_id, permission_id)
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

-- auditor_admin — users.view, roles.view, dashboard.view, short_urls.view_all
INSERT INTO role_permissions (role_id, permission_id)
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

-- Перенос существующих пользователей на роль admin
UPDATE users
SET role_id = (SELECT id FROM roles WHERE name = 'admin')
WHERE role_id IS NULL;
