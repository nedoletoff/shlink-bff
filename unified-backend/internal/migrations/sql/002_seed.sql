-- Начальные данные: разрешения, роли, назначения

-- Разрешения (базовый набор, детальные будут добавлены в следующей миграции)
INSERT INTO permissions (id, name, description) VALUES
    (gen_random_uuid(), 'dashboard.view',     'Просмотр дашборда'),
    (gen_random_uuid(), 'short_urls.create',  'Создание коротких ссылок'),
    (gen_random_uuid(), 'short_urls.update',  'Редактирование ссылок'),
    (gen_random_uuid(), 'short_urls.delete',  'Удаление ссылок'),
    (gen_random_uuid(), 'short_urls.view_all','Просмотр всех чужих ссылок'),
    (gen_random_uuid(), 'users.view',         'Просмотр списка пользователей'),
    (gen_random_uuid(), 'users.manage',       'Управление пользователями'),
    (gen_random_uuid(), 'roles.view',         'Просмотр ролей'),
    (gen_random_uuid(), 'roles.manage',       'Управление ролями и разрешениями'),
    (gen_random_uuid(), 'system.config',      'Изменение конфигурации системы'),
    (gen_random_uuid(), 'system.config.view', 'Просмотр конфигурации системы')
ON CONFLICT (name) DO NOTHING;

-- Роли
INSERT INTO roles (id, name, description) VALUES
    (gen_random_uuid(), 'admin',        'Полный доступ ко всем функциям'),
    (gen_random_uuid(), 'viewer',       'Базовый доступ: просмотр и управление своими ссылками'),
    (gen_random_uuid(), 'auditor_admin','Аудит: просмотр пользователей/ролей, чужих ссылок, без создания')
ON CONFLICT (name) DO NOTHING;

-- Назначения: admin – все разрешения
INSERT INTO role_permissions_v2 (role_id, permission_id)
SELECT r.id, p.id
FROM roles r CROSS JOIN permissions p
WHERE r.name = 'admin'
ON CONFLICT DO NOTHING;

-- viewer – только dashboard.view и short_urls.create/update/delete (но без view_all)
INSERT INTO role_permissions_v2 (role_id, permission_id)
SELECT r.id, p.id
FROM roles r JOIN permissions p ON p.name IN (
    'dashboard.view',
    'short_urls.create',
    'short_urls.update',
    'short_urls.delete',
    'system.config.view'
)
WHERE r.name = 'viewer'
ON CONFLICT DO NOTHING;

-- auditor_admin – только просмотр
INSERT INTO role_permissions_v2 (role_id, permission_id)
SELECT r.id, p.id
FROM roles r JOIN permissions p ON p.name IN (
    'dashboard.view',
    'short_urls.view_all',
    'users.view',
    'roles.view',
    'system.config.view'
)
WHERE r.name = 'auditor_admin'
ON CONFLICT DO NOTHING;

-- Перенос существующих пользователей: если role_text = 'admin' или role_id уже есть, оставляем.
-- Для остальных назначаем роль viewer.
UPDATE users
SET role_id = (SELECT id FROM roles WHERE name = 'viewer')
WHERE role_id IS NULL;
