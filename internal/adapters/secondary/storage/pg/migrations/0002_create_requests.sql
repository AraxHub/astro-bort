-- Таблица запросов (финальная версия со всеми полями)
CREATE TABLE IF NOT EXISTS requests (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,                        -- FK на tg_users.id
    bot_id VARCHAR(255) NOT NULL DEFAULT '',     -- ID бота для роутинга ответов из RAG
    tg_update_id BIGINT UNIQUE,                   -- update_id для дедупликации
    request_text TEXT NOT NULL,                   -- текст запроса от юзера
    response TEXT NOT NULL DEFAULT '',            -- ответ от RAG
    request_type VARCHAR(50) NOT NULL DEFAULT 'user', -- тип запроса ('user', 'push_*', etc.)
    created_at TIMESTAMP DEFAULT NOW(),
    
    FOREIGN KEY (user_id) REFERENCES tg_users(id) ON DELETE CASCADE
);

-- Индексы для requests
CREATE INDEX IF NOT EXISTS idx_requests_user_id ON requests(user_id);
CREATE INDEX IF NOT EXISTS idx_requests_tg_update_id ON requests(tg_update_id);
CREATE INDEX IF NOT EXISTS idx_requests_created_at ON requests(created_at);
CREATE INDEX IF NOT EXISTS idx_requests_bot_id ON requests(bot_id);
CREATE INDEX IF NOT EXISTS idx_requests_request_type ON requests(request_type);
CREATE INDEX IF NOT EXISTS idx_requests_is_push ON requests(request_type) WHERE request_type LIKE 'push_%';
