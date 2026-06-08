-- migrations/007_fix_user_permissions.sql
-- Включаем can_create_with_custom_slug для роли user.
-- Эта миграция исправляет дефолтный seed из 003, где флаг был false,
-- что блокировало создание ссылок с кастомным slug для обычных пользователей.
-- Используем INSERT ... ON CONFLICT DO UPDATE чтобы миграция была идемпотентной
-- независимо от того, была ли строка изменена вручную через UI.

UPDATE role_permissions
SET    can_create_with_custom_slug = true,
       updated_at                  = NOW()
WHERE  role = 'user'
  AND  can_create_with_custom_slug = false;
