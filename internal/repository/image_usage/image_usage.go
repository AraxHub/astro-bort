package imageUsageRepo

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	ports "github.com/admin/tg-bots/astro-bot/internal/ports/repository"

	"log/slog"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/admin/tg-bots/astro-bot/internal/ports/persistence"
)

type imageUsageColumns struct {
	TableName  string
	ChatID     string
	UsedImages string
	CreatedAt  string
	UpdatedAt  string
}

type Repository struct {
	db      persistence.Persistence
	Log     *slog.Logger
	columns imageUsageColumns
}

// New создаёт новый репозиторий для работы со статистикой использования картинок
func New(db persistence.Persistence, log *slog.Logger) ports.IImageUsageRepo {
	cols := imageUsageColumns{
		TableName:  "image_usage",
		ChatID:     "chat_id",
		UsedImages: "used_images",
		CreatedAt:  "created_at",
		UpdatedAt:  "updated_at",
	}
	return &Repository{
		db:      db,
		Log:     log,
		columns: cols,
	}
}

// allColumns возвращает строку со всеми колонками
func (r *Repository) allColumns() string {
	return fmt.Sprintf("%s, %s, %s, %s",
		r.columns.ChatID,
		r.columns.UsedImages,
		r.columns.CreatedAt,
		r.columns.UpdatedAt)
}

// GetOrCreate получает статистику использования для чата или создаёт новую запись
func (r *Repository) GetOrCreate(ctx context.Context, chatID int64) (*domain.ImageUsage, error) {
	usage, err := r.GetUsage(ctx, chatID)
	if err == nil && usage != nil {
		return usage, nil
	}

	// Если не найдено, создаём новую запись
	now := time.Now()
	usage = &domain.ImageUsage{
		ChatID:     chatID,
		UsedImages: make(map[string]int),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	usedImagesJSON, err := json.Marshal(usage.UsedImages)
	if err != nil {
		r.Log.Error("failed to marshal used_images",
			"error", err,
			"chat_id", chatID)
		return nil, fmt.Errorf("failed to marshal used_images: %w", err)
	}

	query := fmt.Sprintf(`INSERT INTO %s (%s, %s, %s, %s) VALUES ($1, $2, $3, $4)`,
		r.columns.TableName,
		r.columns.ChatID,
		r.columns.UsedImages,
		r.columns.CreatedAt,
		r.columns.UpdatedAt)
	err = r.db.Exec(ctx, query, chatID, usedImagesJSON, now, now)
	if err != nil {
		r.Log.Error("failed to create image_usage",
			"error", err,
			"chat_id", chatID)
		return nil, fmt.Errorf("failed to create image_usage: %w", err)
	}

	r.Log.Debug("image_usage created successfully", "chat_id", chatID)
	return usage, nil
}

// imageUsageRow структура для сканирования из БД (JSONB требует специальной обработки)
type imageUsageRow struct {
	ChatID     int64           `db:"chat_id"`
	UsedImages json.RawMessage `db:"used_images"`
	CreatedAt  time.Time       `db:"created_at"`
	UpdatedAt  time.Time       `db:"updated_at"`
}

// GetUsage получает статистику использования для чата
func (r *Repository) GetUsage(ctx context.Context, chatID int64) (*domain.ImageUsage, error) {
	var row imageUsageRow

	query := fmt.Sprintf(`SELECT %s FROM %s WHERE %s = $1`,
		r.allColumns(),
		r.columns.TableName,
		r.columns.ChatID)
	err := r.db.Get(ctx, &row, query, chatID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Возвращаем nil, если не найдено
		}
		r.Log.Error("failed to get image_usage",
			"error", err,
			"chat_id", chatID)
		return nil, fmt.Errorf("failed to get image_usage: %w", err)
	}

	usage := &domain.ImageUsage{
		ChatID:    row.ChatID,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}

	// Парсим JSONB в map
	if len(row.UsedImages) > 0 {
		usage.UsedImages = make(map[string]int)
		if err := json.Unmarshal(row.UsedImages, &usage.UsedImages); err != nil {
			r.Log.Error("failed to unmarshal used_images",
				"error", err,
				"chat_id", chatID)
			return nil, fmt.Errorf("failed to unmarshal used_images: %w", err)
		}
	} else {
		usage.UsedImages = make(map[string]int)
	}

	return usage, nil
}

// UpdateUsage обновляет статистику использования для чата
func (r *Repository) UpdateUsage(ctx context.Context, chatID int64, usedImages map[string]int) error {
	usedImagesJSON, err := json.Marshal(usedImages)
	if err != nil {
		r.Log.Error("failed to marshal used_images",
			"error", err,
			"chat_id", chatID)
		return fmt.Errorf("failed to marshal used_images: %w", err)
	}

	query := fmt.Sprintf(`UPDATE %s SET %s = $1, %s = $2 WHERE %s = $3`,
		r.columns.TableName,
		r.columns.UsedImages,
		r.columns.UpdatedAt,
		r.columns.ChatID)
	rowsAffected, err := r.db.ExecWithResult(ctx, query, usedImagesJSON, time.Now(), chatID)
	if err != nil {
		r.Log.Error("failed to update image_usage",
			"error", err,
			"chat_id", chatID)
		return fmt.Errorf("failed to update image_usage: %w", err)
	}
	if rowsAffected == 0 {
		r.Log.Warn("image_usage not found for update", "chat_id", chatID)
		return fmt.Errorf("image_usage not found for chat_id: %d", chatID)
	}

	r.Log.Debug("image_usage updated successfully", "chat_id", chatID)
	return nil
}

// IncrementUsage инкрементирует счётчик использования конкретной картинки
func (r *Repository) IncrementUsage(ctx context.Context, chatID int64, filename string) error {
	// Используем PostgreSQL JSONB операторы для атомарного инкремента
	// jsonb_set обновляет значение по ключу, создавая его если не существует
	query := fmt.Sprintf(`UPDATE %s SET 
		%s = jsonb_set(
			COALESCE(%s, '{}'::jsonb),
			$1,
			to_jsonb(COALESCE((%s->>$1)::int, 0) + 1)
		),
		%s = $2
		WHERE %s = $3`,
		r.columns.TableName,
		r.columns.UsedImages,
		r.columns.UsedImages,
		r.columns.UsedImages,
		r.columns.UpdatedAt,
		r.columns.ChatID)

	// jsonb_set требует путь в формате массива: {"filename"}
	keyPath := fmt.Sprintf(`{"%s"}`, filename)
	rowsAffected, err := r.db.ExecWithResult(ctx, query, keyPath, time.Now(), chatID)
	if err != nil {
		r.Log.Error("failed to increment image usage",
			"error", err,
			"chat_id", chatID,
			"filename", filename)
		return fmt.Errorf("failed to increment image usage: %w", err)
	}
	if rowsAffected == 0 {
		r.Log.Warn("image_usage not found for increment", "chat_id", chatID)
		return fmt.Errorf("image_usage not found for chat_id: %d", chatID)
	}

	r.Log.Debug("image usage incremented successfully",
		"chat_id", chatID,
		"filename", filename)
	return nil
}
