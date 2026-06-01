-- migrations/001_init_schema.sql
-- Инициализация схемы БД для python-backend (FastAPI) -- MySQL 8.x edition

-- Установить charset по умолчанию для базы (MySQL 8 использует utf8mb4 по умолчанию)

-- Пользователи: основная таблица идентификаторов
CREATE TABLE IF NOT EXISTS users (
  id           CHAR(36)     NOT NULL PRIMARY KEY DEFAULT (UUID()),
  sub          VARCHAR(512) NOT NULL,
  username     TEXT         NOT NULL,
  email        TEXT         NOT NULL,
  role         VARCHAR(16)  NOT NULL DEFAULT 'user',
  shlink_api_key TEXT       NOT NULL DEFAULT '',
  slug_prefix  VARCHAR(128),
  status       VARCHAR(16)  NOT NULL DEFAULT 'active',
  created_at   DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at   DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  CONSTRAINT ck_users_role   CHECK (role   IN ('admin', 'user')),
  CONSTRAINT ck_users_status CHECK (status IN ('active', 'disabled', 'pending')),
  UNIQUE KEY idx_users_sub (sub(191)),
  KEY idx_users_role   (role),
  KEY idx_users_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Теги пользователей: изоляция тегов (feature: userTagInternalIdEnabled)
CREATE TABLE IF NOT EXISTS user_tags (
  id           CHAR(36)     NOT NULL PRIMARY KEY DEFAULT (UUID()),
  user_id      CHAR(36)     NOT NULL,
  tag_name     VARCHAR(255) NOT NULL,
  internal_id  VARCHAR(255),
  created_at   DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  UNIQUE KEY uq_user_tags_user_tag (user_id, tag_name),
  KEY idx_user_tags_user_id    (user_id),
  KEY idx_user_tags_internal_id (internal_id),
  CONSTRAINT fk_user_tags_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Журнал аудита
CREATE TABLE IF NOT EXISTS audit_logs (
  id           BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
  user_sub     TEXT         NOT NULL,
  username     TEXT,
  role         TEXT,
  action       TEXT         NOT NULL,
  resource     TEXT,
  result       VARCHAR(16)  NOT NULL,
  details      JSON,
  ip_address   VARCHAR(64),
  user_agent   TEXT,
  created_at   DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  CONSTRAINT ck_audit_logs_result CHECK (result IN ('success', 'denied', 'error')),
  KEY idx_audit_logs_user_sub   ((CAST(user_sub AS CHAR(512)))),
  KEY idx_audit_logs_created_at (created_at DESC),
  KEY idx_audit_logs_result     (result)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
