package telegram

import (
	"context"
)

// IClient интерфейс для клиента Telegram API
type IClient interface {
	SendMessage(ctx context.Context, chatID int64, text string) error
	SendMessageWithKeyboard(ctx context.Context, chatID int64, text string, keyboard map[string]interface{}) error
	AnswerCallbackQuery(ctx context.Context, callbackID string, text string, showAlert bool) error
}
