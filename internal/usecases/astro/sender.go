package astro

import (
	"context"
	"fmt"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
)

// sendMessage отправляет сообщение пользователю через Telegram Service
func (s *Service) sendMessage(ctx context.Context, botID domain.BotId, chatID int64, text string) error {
	if err := s.TelegramService.SendMessage(ctx, botID, chatID, text); err != nil {
		s.Log.Error("failed to send message",
			"error", err,
			"bot_id", botID,
			"chat_id", chatID,
		)
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// sendMessageWithMarkdown отправляет сообщение с Markdown форматированием через Telegram Service
func (s *Service) sendMessageWithMarkdown(ctx context.Context, botID domain.BotId, chatID int64, text string) error {
	if err := s.TelegramService.SendMessageWithMarkdown(ctx, botID, chatID, text); err != nil {
		s.Log.Error("failed to send message with markdown",
			"error", err,
			"bot_id", botID,
			"chat_id", chatID,
		)
		return fmt.Errorf("failed to send message with markdown: %w", err)
	}

	return nil
}

// sendMessageWithKeyboard отправляет сообщение с inline клавиатурой через Telegram Service
func (s *Service) sendMessageWithKeyboard(ctx context.Context, botID domain.BotId, chatID int64, text string, keyboard map[string]interface{}) error {
	if err := s.TelegramService.SendMessageWithKeyboard(ctx, botID, chatID, text, keyboard); err != nil {
		s.Log.Error("failed to send message with keyboard",
			"error", err,
			"bot_id", botID,
			"chat_id", chatID,
		)
		return fmt.Errorf("failed to send message with keyboard: %w", err)
	}

	return nil
}

// sendMessageWithKeyboardAndMarkdown отправляет сообщение с inline клавиатурой и Markdown форматированием через Telegram Service
func (s *Service) sendMessageWithKeyboardAndMarkdown(ctx context.Context, botID domain.BotId, chatID int64, text string, keyboard map[string]interface{}) error {
	if err := s.TelegramService.SendMessageWithKeyboardAndMarkdown(ctx, botID, chatID, text, keyboard); err != nil {
		s.Log.Error("failed to send message with keyboard and markdown",
			"error", err,
			"bot_id", botID,
			"chat_id", chatID,
		)
		return fmt.Errorf("failed to send message with keyboard and markdown: %w", err)
	}

	return nil
}
