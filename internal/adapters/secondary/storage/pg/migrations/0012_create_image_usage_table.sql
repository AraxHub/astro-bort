-- Таблица для отслеживания использования картинок по чатам
-- Использует JSONB для компактного хранения счётчиков использования
CREATE TABLE IF NOT EXISTS image_usage (
    chat_id BIGINT PRIMARY KEY,                   -- ID чата из Telegram
    used_images JSONB DEFAULT '{}'::jsonb,        -- JSON объект: {"L1": 3, "L2": 2, "L3": 1}
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- GIN индекс для эффективного поиска по JSONB полю
CREATE INDEX IF NOT EXISTS idx_image_usage_used_images ON image_usage USING GIN (used_images);

-- Индекс для обновления записей
CREATE INDEX IF NOT EXISTS idx_image_usage_updated_at ON image_usage(updated_at);
