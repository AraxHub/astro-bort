-- Добавляем колонку request_type для различения типов запросов
ALTER TABLE requests ADD COLUMN IF NOT EXISTS request_type VARCHAR(50) NOT NULL DEFAULT 'user';

-- Создаём индекс для быстрого поиска по типу запроса
CREATE INDEX IF NOT EXISTS idx_requests_request_type ON requests(request_type);

-- Создаём индекс для поиска пушей (все типы, начинающиеся с 'push_')
CREATE INDEX IF NOT EXISTS idx_requests_is_push ON requests(request_type) WHERE request_type LIKE 'push_%';
