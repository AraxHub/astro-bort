-- Таблица для хранения метаданных картинок из S3
CREATE TABLE IF NOT EXISTS images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    filename VARCHAR(255) NOT NULL UNIQUE,     -- Имя файла в S3 (например, "L1.jpg", "L2.jpg")
    tg_file_id VARCHAR(255) NOT NULL,         -- file_id от Telegram для быстрой отправки
    theme VARCHAR(50),                         -- Тема картинки (например, "love", "career", "money")
    created_at TIMESTAMP DEFAULT NOW()
);

-- Индексы для images
CREATE INDEX IF NOT EXISTS idx_images_filename ON images(filename);
CREATE INDEX IF NOT EXISTS idx_images_tg_file_id ON images(tg_file_id);
CREATE INDEX IF NOT EXISTS idx_images_theme ON images(theme) WHERE theme IS NOT NULL;
