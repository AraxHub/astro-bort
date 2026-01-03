package service

import (
	"context"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
)

// IBotService интерфейс для бизнес-логики любого бота
type IBotService interface {
	HandleCommand(ctx context.Context, botID domain.BotId, user *domain.User, command string, updateID int64) error
	HandleText(ctx context.Context, botID domain.BotId, user *domain.User, text string, updateID int64) error
	GetOrCreateUser(ctx context.Context, botID domain.BotId, tgUser *domain.TelegramUser, chat *domain.Chat) (*domain.User, error)
}
