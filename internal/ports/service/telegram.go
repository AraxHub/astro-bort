package service

import (
	"context"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
)

// ITelegramService интерфейс для отправки сообщений через Telegram
type ITelegramService interface {
	SendMessage(ctx context.Context, botID domain.BotId, chatID int64, text string) error
	SendMessageWithKeyboard(ctx context.Context, botID domain.BotId, chatID int64, text string, keyboard map[string]interface{}) error
	AnswerCallbackQuery(ctx context.Context, botID domain.BotId, callbackID string, text string, showAlert bool) error
}
