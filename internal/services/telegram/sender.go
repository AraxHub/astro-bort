package telegram

import (
	"context"
)

// SendMessage отправляет текстовое сообщение пользователю
// TODO: реализовать через Telegram Client из адаптеров
func (s *Service) SendMessage(ctx context.Context, botID string, chatID int64, text string) error {
	// TODO: использовать Telegram Client для отправки сообщения
	s.Log.Info("sending message",
		"bot_id", botID,
		"chat_id", chatID,
		"text_length", len(text),
	)
	
	// Временная заглушка
	return nil
}

// SendMessageWithKeyboard отправляет сообщение с клавиатурой
// TODO: реализовать через Telegram Client из адаптеров
func (s *Service) SendMessageWithKeyboard(ctx context.Context, botID string, chatID int64, text string, keyboard interface{}) error {
	// TODO: использовать Telegram Client для отправки сообщения с клавиатурой
	s.Log.Info("sending message with keyboard",
		"bot_id", botID,
		"chat_id", chatID,
		"text_length", len(text),
	)
	
	// Временная заглушка
	return nil
}

