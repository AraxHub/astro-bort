-- Таблица пользователей
create table if not exists tg_users (
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
    
    -- Метаданные
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    last_seen_at TIMESTAMP                       -- последняя активность
);

create index if not exists idx_tg_users_tg_id on tg_users(tg_id);
create index if not exists idx_tg_users_chat_id on tg_users(chat_id);
create index if not exists idx_tg_users_birth_datetime on tg_users(birth_datetime) where birth_datetime is not null;

-- Таблица запросов
create table if not exists requests (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,                        -- FK на tg_users.id
    tg_update_id BIGINT UNIQUE,                   -- update_id для дедупликации
    request_text TEXT NOT NULL,                    -- текст запроса от юзера
    created_at TIMESTAMP DEFAULT NOW(),
    
    FOREIGN KEY (user_id) REFERENCES tg_users(id) ON DELETE CASCADE
);

-- Индексы для requests
create index if not exists idx_requests_user_id on requests(user_id);
create index if not exists idx_requests_tg_update_id on requests(tg_update_id);
create index if not exists idx_requests_created_at on requests(created_at);

-- ENUM для типов сущностей
create type entity_type_enum as enum ('request');

-- Универсальная таблица статусов для всех сущностей (event sourcing)
create table if not exists statuses (
    id UUID PRIMARY KEY,
    object_type entity_type_enum NOT NULL,         -- тип сущности: 'request', 'user', 'payment', etc.
    object_id UUID NOT NULL,                       -- ID сущности (request_id, user_id, payment_id, etc.)
    status SMALLINT NOT NULL,                      -- числовой статус
    error_message TEXT,                            -- ошибка (если status = failed)
    metadata JSONB,                                -- дополнительные данные:
                                                   --   для request: kafka_partition, kafka_offset, rag_request_id, rag_response_id, telegram_message_id
                                                   --   для user: natal_chart_fetch_error, payment_info, etc.
    created_at TIMESTAMP DEFAULT NOW()
);

-- Индексы для statuses
create index if not exists idx_statuses_entity on statuses(object_type, object_id);  -- для поиска статусов конкретной сущности
create index if not exists idx_statuses_object_type_status on statuses(object_type, status);  -- для фильтрации по типу и статусу
create index if not exists idx_statuses_created_at on statuses(created_at);  -- для временных запросов
