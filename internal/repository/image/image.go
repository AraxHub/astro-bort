package imageRepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	ports "github.com/admin/tg-bots/astro-bot/internal/ports/repository"

	"log/slog"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/admin/tg-bots/astro-bot/internal/ports/persistence"
)

type imageColumns struct {
	TableName string
	ID        string
	Filename  string
	TgFileID  string
	Theme     string
	CreatedAt string
}

type Repository struct {
	db      persistence.Persistence
	Log     *slog.Logger
	columns imageColumns
}

// New создаёт новый репозиторий для работы с картинками
func New(db persistence.Persistence, log *slog.Logger) ports.IImageRepo {
	cols := imageColumns{
		TableName: "images",
		ID:        "id",
		Filename:  "filename",
		TgFileID:  "tg_file_id",
		Theme:     "theme",
		CreatedAt: "created_at",
	}
	return &Repository{
		db:      db,
		Log:     log,
		columns: cols,
	}
}

// allColumns возвращает строку со всеми колонками
func (r *Repository) allColumns() string {
	return fmt.Sprintf("%s, %s, %s, %s, %s",
		r.columns.ID,
		r.columns.Filename,
		r.columns.TgFileID,
		r.columns.Theme,
		r.columns.CreatedAt)
}

// Create создаёт новую запись о картинке
func (r *Repository) Create(ctx context.Context, image *domain.Image) error {
	query := fmt.Sprintf(`INSERT INTO %s (%s, %s, %s, %s, %s) VALUES ($1, $2, $3, $4, $5)`,
		r.columns.TableName,
		r.columns.ID,
		r.columns.Filename,
		r.columns.TgFileID,
		r.columns.Theme,
		r.columns.CreatedAt)
	err := r.db.Exec(ctx, query,
		image.ID,
		image.Filename,
		image.TgFileID,
		image.Theme,
		image.CreatedAt)
	if err != nil {
		r.Log.Error("failed to create image",
			"error", err,
			"filename", image.Filename,
			"image_id", image.ID)
		return fmt.Errorf("failed to create image: %w", err)
	}
	r.Log.Debug("image created successfully",
		"id", image.ID,
		"filename", image.Filename)
	return nil
}

// GetByFilename получает картинку по имени файла
func (r *Repository) GetByFilename(ctx context.Context, filename string) (*domain.Image, error) {
	var image domain.Image
	query := fmt.Sprintf(`SELECT %s FROM %s WHERE %s = $1`,
		r.allColumns(),
		r.columns.TableName,
		r.columns.Filename)
	err := r.db.Get(ctx, &image, query, filename)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.Log.Debug("image not found", "filename", filename)
			return nil, fmt.Errorf("image not found: %w", err)
		}
		r.Log.Error("failed to get image by filename",
			"error", err,
			"filename", filename)
		return nil, fmt.Errorf("failed to get image by filename: %w", err)
	}
	return &image, nil
}

// GetByTheme получает все картинки определённой темы
func (r *Repository) GetByTheme(ctx context.Context, theme domain.ImageTheme) ([]*domain.Image, error) {
	var images []*domain.Image
	query := fmt.Sprintf(`SELECT %s FROM %s WHERE %s = $1 ORDER BY %s`,
		r.allColumns(),
		r.columns.TableName,
		r.columns.Theme,
		r.columns.Filename)
	err := r.db.Select(ctx, &images, query, theme.String())
	if err != nil {
		r.Log.Error("failed to get images by theme",
			"error", err,
			"theme", theme)
		return nil, fmt.Errorf("failed to get images by theme: %w", err)
	}
	return images, nil
}

// GetTgFileIDByFilename получает telegram_file_id по имени файла
func (r *Repository) GetTgFileIDByFilename(ctx context.Context, filename string) (string, error) {
	var tgFileID string
	query := fmt.Sprintf(`SELECT %s FROM %s WHERE %s = $1`,
		r.columns.TgFileID,
		r.columns.TableName,
		r.columns.Filename)
	err := r.db.Get(ctx, &tgFileID, query, filename)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.Log.Debug("image not found", "filename", filename)
			return "", fmt.Errorf("image not found: %w", err)
		}
		r.Log.Error("failed to get tg_file_id by filename",
			"error", err,
			"filename", filename)
		return "", fmt.Errorf("failed to get tg_file_id by filename: %w", err)
	}
	return tgFileID, nil
}

// GetAll получает все картинки
func (r *Repository) GetAll(ctx context.Context) ([]*domain.Image, error) {
	var images []*domain.Image
	query := fmt.Sprintf(`SELECT %s FROM %s ORDER BY %s, %s`,
		r.allColumns(),
		r.columns.TableName,
		r.columns.Theme,
		r.columns.Filename)
	err := r.db.Select(ctx, &images, query)
	if err != nil {
		r.Log.Error("failed to get all images", "error", err)
		return nil, fmt.Errorf("failed to get all images: %w", err)
	}
	return images, nil
}
