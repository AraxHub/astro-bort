package service

import (
	"context"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
)

// ITelegramService интерфейс для отправки сообщений через Telegram
type ITelegramService interface {
	SendMessage(ctx context.Context, botID domain.BotId, chatID int64, text string) error
	SendMessageWithID(ctx context.Context, botID domain.BotId, chatID int64, text string) (int64, error) // возвращает messageID
	SendMessageWithIDAndHTML(ctx context.Context, botID domain.BotId, chatID int64, text string) (int64, error)
	SendMessageWithMarkdown(ctx context.Context, botID domain.BotId, chatID int64, text string) error
	SendMessageWithKeyboard(ctx context.Context, botID domain.BotId, chatID int64, text string, keyboard map[string]interface{}) error
	SendMessageWithKeyboardAndMarkdown(ctx context.Context, botID domain.BotId, chatID int64, text string, keyboard map[string]interface{}) error
	SendPhoto(ctx context.Context, botID domain.BotId, chatID int64, messageThreadID *int64, photoData []byte, filename string) (string, error) // возвращает file_id
	SendPhotoByFileID(ctx context.Context, botID domain.BotId, chatID int64, fileID string) error
	AnswerCallbackQuery(ctx context.Context, botID domain.BotId, callbackID string, text string, showAlert bool) error
	EditMessageReplyMarkup(ctx context.Context, botID domain.BotId, chatID int64, messageID int64, replyMarkup map[string]interface{}) error
}
