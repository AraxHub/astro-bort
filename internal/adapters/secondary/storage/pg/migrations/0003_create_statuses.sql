-- ENUM для типов сущностей
-- Создаём enum с обоими значениями сразу (если не существует)
-- Если enum уже существует с 'request', значение 'payment' должно быть добавлено отдельной миграцией
CREATE TYPE entity_type_enum AS ENUM ('request', 'payment');

-- Универсальная таблица статусов для всех сущностей (event sourcing)
CREATE TABLE IF NOT EXISTS statuses (
    id UUID PRIMARY KEY,
    object_type entity_type_enum NOT NULL,         -- тип сущности: 'request', 'payment'
    object_id UUID NOT NULL,                       -- ID сущности (request_id, payment_id, etc.)
    status SMALLINT NOT NULL,                      -- числовой статус
    error_message TEXT,                            -- ошибка (если status = failed)
    metadata JSONB,                                -- дополнительные данные:
                                                   --   для request: kafka_partition, kafka_offset, rag_request_id, rag_response_id, telegram_message_id
                                                   --   для payment: phase, stage, context (user_id, product_id, amount, etc.)
    created_at TIMESTAMP DEFAULT NOW()
);

-- Индексы для statuses
CREATE INDEX IF NOT EXISTS idx_statuses_entity ON statuses(object_type, object_id);  -- для поиска статусов конкретной сущности
CREATE INDEX IF NOT EXISTS idx_statuses_object_type_status ON statuses(object_type, status);  -- для фильтрации по типу и статусу
CREATE INDEX IF NOT EXISTS idx_statuses_created_at ON statuses(created_at);  -- для временных запросов
