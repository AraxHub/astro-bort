-- Добавляем bot_id в таблицу payments для учёта платежей по ботам
ALTER TABLE payments ADD COLUMN IF NOT EXISTS bot_id VARCHAR(255) NOT NULL DEFAULT '';

-- Создаём индекс для быстрого поиска по bot_id
CREATE INDEX IF NOT EXISTS idx_payments_bot_id ON payments(bot_id);

-- Составной индекс для поиска платежей пользователя в конкретном боте
CREATE INDEX IF NOT EXISTS idx_payments_bot_id_user_id ON payments(bot_id, user_id);
