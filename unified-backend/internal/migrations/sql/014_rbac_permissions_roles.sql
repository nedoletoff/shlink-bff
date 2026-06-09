-- migrations/014_rbac_permissions_roles.sql
-- Переход к унифицированной RBAC-модели:
-- таблицы permissions, roles (новые), role_permissions (many-to-many),
-- добавление role_id в users.
-- Старое поле role остаётся для обратной совместимости на время перехода.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Набор атомарных действий, которые можно проверять
CREATE TABLE IF NOT EXISTS permissions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT UNIQUE NOT NULL,  -- 'dashboard.view', 'users.manage', …
    description TEXT
);

-- Роли (отдельная таблица, независимая от поля role в users)
CREATE TABLE IF NOT EXISTS roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT UNIQUE NOT NULL,  -- 'admin', 'viewer', 'auditor_admin', …
    description TEXT
);

-- Связь ролей с разрешениями (many-to-many)
CREATE TABLE IF NOT EXISTS role_permissions_v2 (
    role_id       UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- Ссылка пользователя на новую роль
ALTER TABLE users ADD COLUMN IF NOT EXISTS role_id UUID REFERENCES roles(id);

CREATE INDEX IF NOT EXISTS idx_users_role_id ON users (role_id);
CREATE INDEX IF NOT EXISTS idx_rp_v2_role_id ON role_permissions_v2 (role_id);
CREATE INDEX IF NOT EXISTS idx_rp_v2_perm_id ON role_permissions_v2 (permission_id);
