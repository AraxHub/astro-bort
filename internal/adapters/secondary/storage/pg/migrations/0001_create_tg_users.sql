-- Таблица пользователей (финальная версия со всеми полями)
CREATE TABLE IF NOT EXISTS tg_users (
    id UUID PRIMARY KEY,
    tg_id BIGINT UNIQUE NOT NULL,                -- from.id из Telegram (основной ключ для маппинга)
    chat_id BIGINT NOT NULL,                     -- chat.id (для отправки ответа)
    first_name VARCHAR(255) NOT NULL,            -- имя (обязательное)
    last_name VARCHAR(255),                      -- фамилия (опциональное)
    username VARCHAR(255),                       -- username (опциональное)

    -- Дата и время рождения (для натальной карты)
    birth_datetime TIMESTAMP,                    -- дата и время рождения (YYYY-MM-DD HH:MM:SS, опционально)
    birth_place VARCHAR(255),                    -- место рождения (опционально)
    birth_data_set_at TIMESTAMP,                 -- когда установили дату рождения
    birth_data_can_change_until TIMESTAMP,       -- до какого времени можно изменить (24 часа)
    
    -- Натальная карта
    natal_chart JSONB,                          -- натальная карта (если есть)
    natal_chart_fetched_at TIMESTAMP,            -- когда получили натальную карту
    
    -- Платежи и подписки
    is_paid BOOLEAN NOT NULL DEFAULT FALSE,      -- платный доступ
    free_msg_count INTEGER NOT NULL DEFAULT 0,    -- счётчик бесплатных сообщений
    manual_granted BOOLEAN NOT NULL DEFAULT FALSE, -- ручная выдача доступа (админские аккаунты)
    
    -- Метаданные
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    last_seen_at TIMESTAMP                       -- последняя активность
);

-- Индексы для tg_users
CREATE INDEX IF NOT EXISTS idx_tg_users_tg_id ON tg_users(tg_id);
CREATE INDEX IF NOT EXISTS idx_tg_users_chat_id ON tg_users(chat_id);
CREATE INDEX IF NOT EXISTS idx_tg_users_birth_datetime ON tg_users(birth_datetime) WHERE birth_datetime IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_is_paid ON tg_users(is_paid);
CREATE INDEX IF NOT EXISTS idx_users_manual_granted ON tg_users(manual_granted);
