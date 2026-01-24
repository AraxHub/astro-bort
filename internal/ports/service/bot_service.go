package service

import (
	"context"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/google/uuid"
)

// IBotService интерфейс для бизнес-логики любого бота
type IBotService interface {
	HandleCommand(ctx context.Context, botID domain.BotId, user *domain.User, command string, updateID int64) error
	HandleText(ctx context.Context, botID domain.BotId, user *domain.User, text string, updateID int64) error
	GetOrCreateUser(ctx context.Context, botID domain.BotId, tgUser *domain.TelegramUser, chat *domain.Chat) (*domain.User, error)
	HandleRAGResponse(ctx context.Context, requestID uuid.UUID, botID domain.BotId, chatID int64, responseText string) error
	HandleWeeklyForecastCallback(ctx context.Context, botID domain.BotId, user *domain.User) error
}
