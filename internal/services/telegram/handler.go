package telegram

import (
	"context"
	"fmt"
	"strings"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/admin/tg-bots/astro-bot/internal/ports/service"
)

// HandleUpdate обрабатывает входящее обновление от Telegram
// Основной метод для обработки всех типов обновлений
func (s *Service) HandleUpdate(ctx context.Context, botID string, update *domain.Update) error {
	if update == nil {
		return fmt.Errorf("update is nil")
	}

	if update.Message != nil {
		return s.HandleMessage(ctx, botID, update.Message, update.UpdateID)
	}

	// TODO: обработать другие типы обновлений (edited_message, callback_query и т.д.)

	return nil
}

// HandleMessage обрабатывает входящее сообщение
// Только техническая валидация и роутинг к use case
func (s *Service) HandleMessage(ctx context.Context, botID string, message *domain.Message, updateID int64) error {
	if message == nil {
		return fmt.Errorf("message is nil")
	}

	// Проверяем, что сообщение от пользователя (не от бота)
	if message.From == nil || message.From.IsBot {
		s.Log.Debug("ignoring message from bot", "update_id", updateID)
		return nil
	}

	// Получаем use case для этого бота
	botService, ok := s.BotServices[botID]
	if !ok {
		return fmt.Errorf("unknown bot_id: %s", botID)
	}

	// Получаем или создаём пользователя через use case (доменная логика)
	user, err := botService.GetOrCreateUser(ctx, message.From, message.Chat)
	if err != nil {
		s.Log.Error("failed to get or create user",
			"error", err,
			"telegram_user_id", message.From.ID,
			"update_id", updateID,
			"bot_id", botID,
		)
		return fmt.Errorf("failed to get or create user: %w", err)
	}

	// Обрабатываем текст сообщения, если есть
	if message.Text != nil {
		return s.routeTextMessage(ctx, botService, user, *message.Text, updateID)
	}

	return nil
}

// routeTextMessage роутит текстовое сообщение к нужному use case
func (s *Service) routeTextMessage(ctx context.Context, botService service.IBotService, user *domain.User, text string, updateID int64) error {
	// Проверяем, является ли сообщение командой
	if IsCommand(text) {
		command := ParseCommand(text)
		return botService.HandleCommand(ctx, user, command, updateID)
	}

	// Обычное текстовое сообщение
	return botService.HandleText(ctx, user, text, updateID)
}

// ParseCommand извлекает команду из текста
// Например: "/start" или "/start@botname" → "start"
func ParseCommand(text string) string {
	// Убираем ведущий "/"
	text = strings.TrimPrefix(text, "/")

	// Убираем @botname если есть
	if idx := strings.Index(text, "@"); idx != -1 {
		text = text[:idx]
	}

	// Убираем аргументы если есть
	if idx := strings.Index(text, " "); idx != -1 {
		text = text[:idx]
	}

	return text
}

// IsCommand проверяет, является ли текст командой
func IsCommand(text string) bool {
	return len(text) > 0 && text[0] == '/'
}
