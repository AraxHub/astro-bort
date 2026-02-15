-- Добавляем колонку onboarding_count в таблицу tg_users
ALTER TABLE tg_users ADD COLUMN IF NOT EXISTS onboarding_count INTEGER NOT NULL DEFAULT 0;

-- Устанавливаем onboarding_count = 3 для существующих пользователей (онбординг пройден)
UPDATE tg_users SET onboarding_count = 3 WHERE onboarding_count = 0;

-- Индекс для быстрого поиска пользователей в онбординге (опционально, если нужно)
CREATE INDEX IF NOT EXISTS idx_tg_users_onboarding_count ON tg_users(onboarding_count) WHERE onboarding_count < 3;
