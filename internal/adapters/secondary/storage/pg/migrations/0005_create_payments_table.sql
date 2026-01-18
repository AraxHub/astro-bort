-- Создание таблицы payments для хранения платежей
CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES tg_users(id) ON DELETE CASCADE,
    amount BIGINT NOT NULL, -- количество звёзд (или копеек для других валют)
    currency VARCHAR(10) NOT NULL DEFAULT 'XTR', -- "XTR" для Stars, "RUB" для рублей и т.д.
    method VARCHAR(50) NOT NULL, -- способ оплаты: "telegram_stars", "yookassa" и т.д.
    provider_id VARCHAR(255) NOT NULL, -- ID в системе провайдера (invoice_id для Stars)
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, succeeded, failed, refunded
    product_id VARCHAR(100) NOT NULL, -- что куплено (например, "premium_access")
    product_title VARCHAR(255) NOT NULL, -- название продукта для отображения
    metadata JSONB DEFAULT '{}', -- дополнительные данные (payload, telegram_payment_charge_id и т.д.)
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    succeeded_at TIMESTAMP,
    failed_at TIMESTAMP,
    error_message TEXT
);

-- Индексы для быстрого поиска
CREATE INDEX IF NOT EXISTS idx_payments_user_id ON payments(user_id);
CREATE INDEX IF NOT EXISTS idx_payments_provider_id ON payments(provider_id);
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status);
CREATE INDEX IF NOT EXISTS idx_payments_created_at ON payments(created_at);
-- Индекс для поиска по payload в metadata (функциональный B-tree индекс, так как GIN не работает с TEXT напрямую)
CREATE INDEX IF NOT EXISTS idx_payments_metadata_payload ON payments ((metadata->>'payload'));
