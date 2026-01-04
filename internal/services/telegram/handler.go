package telegram

import (
	"context"
	"fmt"
	"strings"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/admin/tg-bots/astro-bot/internal/ports/service"
)

// HandleUpdate Основной метод для обработки всех типов обновлений
func (s *Service) HandleUpdate(ctx context.Context, botID domain.BotId, update *domain.Update) error {
	if update == nil {
		return fmt.Errorf("update is nil")
	}

	if update.Message != nil {
		return s.HandleMessage(ctx, botID, update.Message, update.UpdateID)
	}

	return nil
}

// HandleMessage обрабатывает входящее сообщение - роутинг в usecase
func (s *Service) HandleMessage(ctx context.Context, botID domain.BotId, message *domain.Message, updateID int64) error {
	if message == nil {
		return fmt.Errorf("message is nil")
	}

	if message.From == nil || message.From.IsBot {
		s.Log.Debug("ignoring message from bot", "update_id", updateID)
		return nil
	}

	if message.Chat != nil && message.Chat.Type != "private" {
		s.Log.Warn("ignoring message from group/chat",
			"update_id", updateID,
			"chat_type", message.Chat.Type,
			"chat_id", message.Chat.ID,
		)
		return nil
	}

	// Определяем botType из botID через маппинг
	botType, err := s.GetBotType(botID)
	if err != nil {
		return fmt.Errorf("failed to get bot_type for bot_id %s: %w", botID, err)
	}

	botService, ok := s.BotTypeToUsecase[botType]
	if !ok {
		return fmt.Errorf("unknown bot_type: %s", botType)
	}

	// Получаем или создаём пользователя через use case
	user, err := botService.GetOrCreateUser(ctx, botID, message.From, message.Chat)
	if err != nil {
		s.Log.Error("failed to get or create user",
			"error", err,
			"telegram_user_id", message.From.ID,
			"update_id", updateID,
			"bot_id", botID,
		)
		return fmt.Errorf("failed to get or create user: %w", err)
	}

	if message.Text != nil {
		return s.routeTextMessage(ctx, botID, botService, user, *message.Text, updateID)
	}

	return nil
}

// routeTextMessage роутит в команду/текст
func (s *Service) routeTextMessage(ctx context.Context, botID domain.BotId, botService service.IBotService, user *domain.User, text string, updateID int64) error {
	if IsCommand(text) {
		command := ParseCommand(text)
		return botService.HandleCommand(ctx, botID, user, command, updateID)
	}

	return botService.HandleText(ctx, botID, user, text, updateID)
}

func ParseCommand(text string) string {
	text = strings.TrimPrefix(text, "/")

	if idx := strings.Index(text, "@"); idx != -1 {
		text = text[:idx]
	}

	if idx := strings.Index(text, " "); idx != -1 {
		text = text[:idx]
	}

	return text
}

func IsCommand(text string) bool {
	return len(text) > 0 && text[0] == '/'
}
