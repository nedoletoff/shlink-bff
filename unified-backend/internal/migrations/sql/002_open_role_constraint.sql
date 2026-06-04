-- migrations/002_open_role_constraint.sql
-- Снимаем жёсткий CHECK (role IN ('admin', 'user')) чтобы поддерживать
-- произвольные имена ролей из ROLE_GROUPS.

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
