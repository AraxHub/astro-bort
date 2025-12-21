package astro

import (
	"context"
	"fmt"
)

// sendMessage отправляет сообщение пользователю через Telegram Client
func (s *Service) sendMessage(ctx context.Context, chatID int64, text string) error {
	if err := s.TelegramClient.SendMessage(ctx, chatID, text); err != nil {
		s.Log.Error("failed to send message",
			"error", err,
			"chat_id", chatID,
		)
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// sendMessageWithKeyboard отправляет сообщение с inline клавиатурой
func (s *Service) sendMessageWithKeyboard(ctx context.Context, chatID int64, text string, keyboard map[string]interface{}) error {
	if err := s.TelegramClient.SendMessageWithKeyboard(ctx, chatID, text, keyboard); err != nil {
		s.Log.Error("failed to send message with keyboard",
			"error", err,
			"chat_id", chatID,
		)
		return fmt.Errorf("failed to send message with keyboard: %w", err)
	}

	return nil
}
