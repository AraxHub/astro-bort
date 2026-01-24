-- Добавляем поле для ручного выдачи платного доступа (админские аккаунты)
ALTER TABLE tg_users 
    ADD COLUMN IF NOT EXISTS manual_granted BOOLEAN NOT NULL DEFAULT FALSE;

-- Индекс для быстрого поиска пользователей с ручным доступом
CREATE INDEX IF NOT EXISTS idx_users_manual_granted ON tg_users(manual_granted);
