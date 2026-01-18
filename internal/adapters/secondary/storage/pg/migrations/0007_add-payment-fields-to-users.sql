-- Добавляем поля для платных подписок и счётчика бесплатных сообщений
ALTER TABLE tg_users 
    ADD COLUMN IF NOT EXISTS is_paid BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS free_msg_count INTEGER NOT NULL DEFAULT 0;

-- Индекс для быстрого поиска платных пользователей
CREATE INDEX IF NOT EXISTS idx_users_is_paid ON tg_users(is_paid);
