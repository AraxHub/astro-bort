-- Добавляем bot_id в таблицу requests для роутинга ответов из RAG
ALTER TABLE requests ADD COLUMN IF NOT EXISTS bot_id VARCHAR(255) NOT NULL DEFAULT '';

-- Создаём индекс для быстрого поиска по bot_id
CREATE INDEX IF NOT EXISTS idx_requests_bot_id ON requests(bot_id);

-- Удаляем DEFAULT после добавления колонки (если нужно)
-- ALTER TABLE requests ALTER COLUMN bot_id DROP DEFAULT;
