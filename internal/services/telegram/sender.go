package telegram

import (
	"context"
	"fmt"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
)

// SendMessage отправляет текстовое сообщение пользователю
func (s *Service) SendMessage(ctx context.Context, botID domain.BotId, chatID int64, text string) error {
	_, err := s.SendMessageWithID(ctx, botID, chatID, text)
	return err
}

// SendMessageWithID отправляет текстовое сообщение пользователю и возвращает messageID
func (s *Service) SendMessageWithID(ctx context.Context, botID domain.BotId, chatID int64, text string) (int64, error) {
	client, ok := s.TelegramClients[botID]
	if !ok {
		return 0, fmt.Errorf("telegram client not found for bot_id: %s", botID)
	}

	messageID, err := client.SendMessageWithID(ctx, chatID, text)
	if err != nil {
		return 0, fmt.Errorf("failed to send message: %w", err)
	}

	return messageID, nil
}

// SendMessageWithIDAndHTML отправляет текстовое сообщение с HTML форматированием пользователю и возвращает messageID
func (s *Service) SendMessageWithIDAndHTML(ctx context.Context, botID domain.BotId, chatID int64, text string) (int64, error) {
	client, ok := s.TelegramClients[botID]
	if !ok {
		return 0, fmt.Errorf("telegram client not found for bot_id: %s", botID)
	}

	messageID, err := client.SendMessageWithIDAndHTML(ctx, chatID, text)
	if err != nil {
		return 0, fmt.Errorf("failed to send message with HTML: %w", err)
	}

	return messageID, nil
}

// SendMessageWithMarkdown отправляет текстовое сообщение с Markdown форматированием
func (s *Service) SendMessageWithMarkdown(ctx context.Context, botID domain.BotId, chatID int64, text string) error {
	client, ok := s.TelegramClients[botID]
	if !ok {
		return fmt.Errorf("telegram client not found for bot_id: %s", botID)
	}

	if err := client.SendMessageWithMarkdown(ctx, chatID, text); err != nil {
		return fmt.Errorf("failed to send message with markdown: %w", err)
	}

	return nil
}

// SendMessageWithKeyboard отправляет сообщение с клавиатурой
func (s *Service) SendMessageWithKeyboard(ctx context.Context, botID domain.BotId, chatID int64, text string, keyboard map[string]interface{}) error {
	client, ok := s.TelegramClients[botID]
	if !ok {
		return fmt.Errorf("telegram client not found for bot_id: %s", botID)
	}

	if err := client.SendMessageWithKeyboard(ctx, chatID, text, keyboard); err != nil {
		return fmt.Errorf("failed to send message with keyboard: %w", err)
	}

	return nil
}

// SendMessageWithKeyboardAndMarkdown отправляет сообщение с клавиатурой и Markdown форматированием
func (s *Service) SendMessageWithKeyboardAndMarkdown(ctx context.Context, botID domain.BotId, chatID int64, text string, keyboard map[string]interface{}) error {
	client, ok := s.TelegramClients[botID]
	if !ok {
		return fmt.Errorf("telegram client not found for bot_id: %s", botID)
	}

	if err := client.SendMessageWithKeyboardAndMarkdown(ctx, chatID, text, keyboard); err != nil {
		return fmt.Errorf("failed to send message with keyboard and markdown: %w", err)
	}

	return nil
}

// AnswerCallbackQuery отправляет ответ на callback query
func (s *Service) AnswerCallbackQuery(ctx context.Context, botID domain.BotId, callbackID string, text string, showAlert bool) error {
	client, ok := s.TelegramClients[botID]
	if !ok {
		return fmt.Errorf("telegram client not found for bot_id: %s", botID)
	}

	if err := client.AnswerCallbackQuery(ctx, callbackID, text, showAlert); err != nil {
		return fmt.Errorf("failed to answer callback query: %w", err)
	}

	return nil
}

// SendPhoto отправляет фото в чат и возвращает file_id
func (s *Service) SendPhoto(ctx context.Context, botID domain.BotId, chatID int64, messageThreadID *int64, photoData []byte, filename string) (string, error) {
	client, ok := s.TelegramClients[botID]
	if !ok {
		return "", fmt.Errorf("telegram client not found for bot_id: %s", botID)
	}

	fileID, err := client.SendPhoto(ctx, chatID, messageThreadID, photoData, filename)
	if err != nil {
		return "", fmt.Errorf("failed to send photo: %w", err)
	}

	return fileID, nil
}

// SendPhotoByFileID отправляет фото в чат используя уже существующий file_id
func (s *Service) SendPhotoByFileID(ctx context.Context, botID domain.BotId, chatID int64, fileID string) error {
	client, ok := s.TelegramClients[botID]
	if !ok {
		return fmt.Errorf("telegram client not found for bot_id: %s", botID)
	}

	if err := client.SendPhotoByFileID(ctx, chatID, fileID); err != nil {
		return fmt.Errorf("failed to send photo by file_id: %w", err)
	}

	return nil
}

// EditMessageReplyMarkup редактирует reply_markup сообщения (убирает кнопки, если передать пустой reply_markup)
func (s *Service) EditMessageReplyMarkup(ctx context.Context, botID domain.BotId, chatID int64, messageID int64, replyMarkup map[string]interface{}) error {
	client, ok := s.TelegramClients[botID]
	if !ok {
		return fmt.Errorf("telegram client not found for bot_id: %s", botID)
	}

	if err := client.EditMessageReplyMarkup(ctx, chatID, messageID, replyMarkup); err != nil {
		return fmt.Errorf("failed to edit message reply markup: %w", err)
	}

	return nil
}
