package repository

import (
	"context"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
)

// IImageUsageRepo интерфейс для работы со статистикой использования картинок
type IImageUsageRepo interface {
	GetUsage(ctx context.Context, chatID int64) (*domain.ImageUsage, error)
	Create(ctx context.Context, usage *domain.ImageUsage) error
	UpdateUsage(ctx context.Context, chatID int64, usedImages map[string]int) error
	IncrementUsage(ctx context.Context, chatID int64, filename string) error
}
