package telegram

import (
	"context"
	"fmt"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
)

// SendMessage отправляет текстовое сообщение пользователю
func (s *Service) SendMessage(ctx context.Context, botID domain.BotId, chatID int64, text string) error {
	client, ok := s.TelegramClients[botID]
	if !ok {
		return fmt.Errorf("telegram client not found for bot_id: %s", botID)
	}

	if err := client.SendMessage(ctx, chatID, text); err != nil {
		s.Log.Error("failed to send message",
			"error", err,
			"bot_id", botID,
			"chat_id", chatID,
		)
		return fmt.Errorf("failed to send message: %w", err)
	}

	s.Log.Debug("message sent successfully",
		"bot_id", botID,
		"chat_id", chatID,
	)
	return nil
}

// SendMessageWithMarkdown отправляет текстовое сообщение с Markdown форматированием
func (s *Service) SendMessageWithMarkdown(ctx context.Context, botID domain.BotId, chatID int64, text string) error {
	client, ok := s.TelegramClients[botID]
	if !ok {
		return fmt.Errorf("telegram client not found for bot_id: %s", botID)
	}

	if err := client.SendMessageWithMarkdown(ctx, chatID, text); err != nil {
		s.Log.Error("failed to send message with markdown",
			"error", err,
			"bot_id", botID,
			"chat_id", chatID,
		)
		return fmt.Errorf("failed to send message with markdown: %w", err)
	}

	s.Log.Debug("message with markdown sent successfully",
		"bot_id", botID,
		"chat_id", chatID,
	)
	return nil
}

// SendMessageWithKeyboard отправляет сообщение с клавиатурой
func (s *Service) SendMessageWithKeyboard(ctx context.Context, botID domain.BotId, chatID int64, text string, keyboard map[string]interface{}) error {
	client, ok := s.TelegramClients[botID]
	if !ok {
		return fmt.Errorf("telegram client not found for bot_id: %s", botID)
	}

	if err := client.SendMessageWithKeyboard(ctx, chatID, text, keyboard); err != nil {
		s.Log.Error("failed to send message with keyboard",
			"error", err,
			"bot_id", botID,
			"chat_id", chatID,
		)
		return fmt.Errorf("failed to send message with keyboard: %w", err)
	}

	s.Log.Debug("message with keyboard sent successfully",
		"bot_id", botID,
		"chat_id", chatID,
	)
	return nil
}

// AnswerCallbackQuery отправляет ответ на callback query
func (s *Service) AnswerCallbackQuery(ctx context.Context, botID domain.BotId, callbackID string, text string, showAlert bool) error {
	client, ok := s.TelegramClients[botID]
	if !ok {
		return fmt.Errorf("telegram client not found for bot_id: %s", botID)
	}

	if err := client.AnswerCallbackQuery(ctx, callbackID, text, showAlert); err != nil {
		s.Log.Error("failed to answer callback query",
			"error", err,
			"bot_id", botID,
			"callback_id", callbackID,
		)
		return fmt.Errorf("failed to answer callback query: %w", err)
	}

	s.Log.Debug("callback query answered successfully",
		"bot_id", botID,
		"callback_id", callbackID,
	)
	return nil
}
