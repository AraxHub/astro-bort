package repository

import (
	"context"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
)

// IImageRepo интерфейс для работы с метаданными картинок
type IImageRepo interface {
	Create(ctx context.Context, image *domain.Image) error
	GetByFilename(ctx context.Context, filename string) (*domain.Image, error)
	GetByTheme(ctx context.Context, theme domain.ImageTheme) ([]*domain.Image, error)
	GetTgFileIDByFilename(ctx context.Context, filename string) (string, error)
	GetAll(ctx context.Context) ([]*domain.Image, error)
}
