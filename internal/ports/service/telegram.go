package service

import (
	"context"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
)

// ITelegramService интерфейс для отправки сообщений через Telegram
type ITelegramService interface {
	SendMessage(ctx context.Context, botID domain.BotId, chatID int64, text string) error
	SendMessageWithID(ctx context.Context, botID domain.BotId, chatID int64, text string) (int64, error) // возвращает messageID
	SendMessageWithMarkdown(ctx context.Context, botID domain.BotId, chatID int64, text string) error
	SendMessageWithKeyboard(ctx context.Context, botID domain.BotId, chatID int64, text string, keyboard map[string]interface{}) error
	AnswerCallbackQuery(ctx context.Context, botID domain.BotId, callbackID string, text string, showAlert bool) error
	EditMessageReplyMarkup(ctx context.Context, botID domain.BotId, chatID int64, messageID int64, replyMarkup map[string]interface{}) error
}
