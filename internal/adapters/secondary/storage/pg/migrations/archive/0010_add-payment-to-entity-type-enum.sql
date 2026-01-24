-- Добавляем 'payment' в enum entity_type_enum для поддержки статусов платежей
-- ВАЖНО: ALTER TYPE ... ADD VALUE не может выполняться внутри транзакции в PostgreSQL < 12
-- Эта миграция выполняется вне транзакции (см. маркер NO_TRANSACTION)
-- NO_TRANSACTION

-- Проверяем и добавляем значение 'payment' в enum
DO $$ 
BEGIN
    -- Проверяем, существует ли уже значение 'payment' в enum
    IF NOT EXISTS (
        SELECT 1 
        FROM pg_enum 
        WHERE enumlabel = 'payment' 
        AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'entity_type_enum')
    ) THEN
        -- Добавляем значение 'payment' в enum
        ALTER TYPE entity_type_enum ADD VALUE 'payment';
    END IF;
END $$;
