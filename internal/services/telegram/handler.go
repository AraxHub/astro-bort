package telegram

import (
	"context"
	"fmt"
	"strings"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/admin/tg-bots/astro-bot/internal/ports/service"
	"github.com/google/uuid"
)

// HandleUpdate Основной метод для обработки всех типов обновлений
func (s *Service) HandleUpdate(ctx context.Context, botID domain.BotId, update *domain.Update) error {
	if update == nil {
		s.Log.Error("update is nil")
		return fmt.Errorf("update is nil")
	}

	// Обрабатываем pre_checkout_query (для платежей Stars)
	if update.PreCheckoutQuery != nil {
		return s.HandlePreCheckoutQuery(ctx, botID, update.PreCheckoutQuery)
	}

	// Обрабатываем callback_query (для inline-кнопок)
	if update.CallbackQuery != nil {
		return s.HandleCallbackQuery(ctx, botID, update.CallbackQuery)
	}

	if update.Message != nil {
		// Обрабатываем successful_payment (для платежей Stars)
		if update.Message.SuccessfulPayment != nil {
			return s.HandleSuccessfulPayment(ctx, botID, update.Message)
		}

		return s.HandleMessage(ctx, botID, update.Message, update.UpdateID)
	}

	return nil
}

// HandleCallbackQuery обрабатывает callback query от inline-кнопок
func (s *Service) HandleCallbackQuery(ctx context.Context, botID domain.BotId, callbackQuery *domain.CallbackQuery) error {
	if callbackQuery == nil || callbackQuery.From == nil {
		s.Log.Error("callback_query is nil or has no from")
		return fmt.Errorf("invalid callback_query")
	}

	if callbackQuery.Data == nil {
		s.Log.Warn("callback_query has no data", "callback_id", callbackQuery.ID)
		// Отвечаем на callback без текста, чтобы убрать индикатор загрузки
		if err := s.AnswerCallbackQuery(ctx, botID, callbackQuery.ID, "", false); err != nil {
			s.Log.Warn("failed to answer callback query", "error", err)
		}
		return nil
	}

	botType, err := s.GetBotType(botID)
	if err != nil {
		return fmt.Errorf("failed to get bot_type for bot_id %s: %w", botID, err)
	}

	botService, ok := s.BotTypeToUsecase[botType]
	if !ok {
		return fmt.Errorf("unknown bot_type: %s", botType)
	}

	// Получаем или создаём пользователя
	user, err := botService.GetOrCreateUser(ctx, botID, callbackQuery.From, callbackQuery.Message.Chat)
	if err != nil {
		return domain.WrapBusinessError(fmt.Errorf("failed to get or create user: %w", err))
	}

	// Роутинг callback по data
	callbackData := *callbackQuery.Data
	if strings.HasPrefix(callbackData, "weekly_forecast:") {
		return s.handleWeeklyForecastCallback(ctx, botID, callbackQuery, user, callbackData)
	}

	if strings.HasPrefix(callbackData, "premium_limit_pay:") {
		return s.handlePremiumLimitPaymentCallback(ctx, botID, callbackQuery, user, callbackData)
	}

	// Неизвестный callback - отвечаем без текста
	if err := s.AnswerCallbackQuery(ctx, botID, callbackQuery.ID, "", false); err != nil {
		s.Log.Warn("failed to answer callback query", "error", err)
	}

	return nil
}

// handleWeeklyForecastCallback обрабатывает callback для недельного прогноза
func (s *Service) handleWeeklyForecastCallback(ctx context.Context, botID domain.BotId, callbackQuery *domain.CallbackQuery, user *domain.User, callbackData string) error {
	// Парсим user_id из callback_data (формат: "weekly_forecast:{user_id}")
	parts := strings.Split(callbackData, ":")
	if len(parts) != 2 {
		s.Log.Warn("invalid weekly_forecast callback data format", "data", callbackData)
		if err := s.AnswerCallbackQuery(ctx, botID, callbackQuery.ID, "Ошибка обработки запроса", false); err != nil {
			s.Log.Warn("failed to answer callback query", "error", err)
		}
		return nil
	}

	callbackUserID, err := uuid.Parse(parts[1])
	if err != nil {
		s.Log.Warn("failed to parse user_id from callback data", "error", err, "data", callbackData)
		if err := s.AnswerCallbackQuery(ctx, botID, callbackQuery.ID, "Ошибка обработки запроса", false); err != nil {
			s.Log.Warn("failed to answer callback query", "error", err)
		}
		return nil
	}

	// Проверяем, что callback от того же пользователя
	if user.ID != callbackUserID {
		s.Log.Warn("callback user mismatch",
			"callback_user_id", callbackUserID,
			"actual_user_id", user.ID)
		if err := s.AnswerCallbackQuery(ctx, botID, callbackQuery.ID, "Этот прогноз не для вас", false); err != nil {
			s.Log.Warn("failed to answer callback query", "error", err)
		}
		return nil
	}

	// Отвечаем на callback (убираем индикатор загрузки)
	if err := s.AnswerCallbackQuery(ctx, botID, callbackQuery.ID, "", false); err != nil {
		s.Log.Warn("failed to answer callback query", "error", err)
	}

	// Получаем bot service для обработки
	botType, err := s.GetBotType(botID)
	if err != nil {
		return fmt.Errorf("failed to get bot_type: %w", err)
	}

	botService, ok := s.BotTypeToUsecase[botType]
	if !ok {
		return fmt.Errorf("unknown bot_type: %s", botType)
	}

	// Получаем message_id и chat_id из callback query
	if callbackQuery.Message == nil {
		s.Log.Warn("callback query has no message",
			"callback_id", callbackQuery.ID,
			"user_id", user.ID)
		if err := s.AnswerCallbackQuery(ctx, botID, callbackQuery.ID, "Ошибка обработки запроса", false); err != nil {
			s.Log.Warn("failed to answer callback query", "error", err)
		}
		return nil
	}

	messageID := callbackQuery.Message.MessageID
	var chatID int64
	if callbackQuery.Message.Chat != nil {
		chatID = callbackQuery.Message.Chat.ID
	} else {
		s.Log.Warn("callback query message has no chat",
			"callback_id", callbackQuery.ID,
			"user_id", user.ID,
			"message_id", messageID)
		if err := s.AnswerCallbackQuery(ctx, botID, callbackQuery.ID, "Ошибка обработки запроса", false); err != nil {
			s.Log.Warn("failed to answer callback query", "error", err)
		}
		return nil
	}

	// Вызываем метод обработки callback в usecase
	if err := botService.HandleWeeklyForecastCallback(ctx, botID, user, messageID, chatID); err != nil {
		return domain.WrapBusinessError(fmt.Errorf("failed to handle weekly forecast callback: %w", err))
	}

	return nil
}

// handlePremiumLimitPaymentCallback обрабатывает callback для кнопки "Заплатить" в Premium Limit Push
func (s *Service) handlePremiumLimitPaymentCallback(ctx context.Context, botID domain.BotId, callbackQuery *domain.CallbackQuery, user *domain.User, callbackData string) error {
	// Парсим user_id из callback_data (формат: "premium_limit_pay:{user_id}")
	parts := strings.Split(callbackData, ":")
	if len(parts) != 2 {
		s.Log.Warn("invalid premium_limit_pay callback data format", "data", callbackData)
		if err := s.AnswerCallbackQuery(ctx, botID, callbackQuery.ID, "Ошибка обработки запроса", false); err != nil {
			s.Log.Warn("failed to answer callback query", "error", err)
		}
		return nil
	}

	callbackUserID, err := uuid.Parse(parts[1])
	if err != nil {
		s.Log.Warn("failed to parse user_id from callback data", "error", err, "data", callbackData)
		if err := s.AnswerCallbackQuery(ctx, botID, callbackQuery.ID, "Ошибка обработки запроса", false); err != nil {
			s.Log.Warn("failed to answer callback query", "error", err)
		}
		return nil
	}

	// Проверяем, что callback от того же пользователя
	if user.ID != callbackUserID {
		s.Log.Warn("callback user mismatch",
			"callback_user_id", callbackUserID,
			"actual_user_id", user.ID)
		if err := s.AnswerCallbackQuery(ctx, botID, callbackQuery.ID, "Этот запрос не для вас", false); err != nil {
			s.Log.Warn("failed to answer callback query", "error", err)
		}
		return nil
	}

	// Отвечаем на callback (убираем индикатор загрузки)
	if err := s.AnswerCallbackQuery(ctx, botID, callbackQuery.ID, "", false); err != nil {
		s.Log.Warn("failed to answer callback query", "error", err)
	}

	// Получаем bot service для обработки
	botType, err := s.GetBotType(botID)
	if err != nil {
		return fmt.Errorf("failed to get bot_type: %w", err)
	}

	botService, ok := s.BotTypeToUsecase[botType]
	if !ok {
		return fmt.Errorf("unknown bot_type: %s", botType)
	}

	// Вызываем метод обработки callback в usecase
	if err := botService.HandlePremiumLimitPaymentCallback(ctx, botID, user); err != nil {
		return domain.WrapBusinessError(fmt.Errorf("failed to handle premium limit payment callback: %w", err))
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
		s.Log.Debug("ignoring message from group/chat",
			"update_id", updateID,
			"chat_type", message.Chat.Type,
			"chat_id", message.Chat.ID,
		)
		return nil
	}

	botType, err := s.GetBotType(botID)
	if err != nil {
		return fmt.Errorf("failed to get bot_type for bot_id %s: %w", botID, err)
	}

	botService, ok := s.BotTypeToUsecase[botType]
	if !ok {
		return fmt.Errorf("unknown bot_type: %s", botType)
	}

	user, err := botService.GetOrCreateUser(ctx, botID, message.From, message.Chat)
	if err != nil {
		return domain.WrapBusinessError(fmt.Errorf("failed to get or create user: %w", err))
	}

	if message.Text != nil {
		err := s.routeTextMessage(ctx, botID, botService, user, *message.Text, updateID)
		if err != nil {
			return domain.WrapBusinessError(err)
		}
		return nil
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

// HandleRAGResponse обрабатывает ответ от RAG - роутинг в usecase
func (s *Service) HandleRAGResponse(ctx context.Context, requestID uuid.UUID, botID domain.BotId, chatID int64, responseText string) error {
	botType, err := s.GetBotType(botID)
	if err != nil {
		return fmt.Errorf("failed to get bot_type [bot_id=%s, request_id=%s]: %w",
			botID, requestID, err)
	}

	botService, ok := s.BotTypeToUsecase[botType]
	if !ok {
		return fmt.Errorf("unknown bot_type [bot_type=%s, bot_id=%s, request_id=%s]",
			botType, botID, requestID)
	}

	if err = botService.HandleRAGResponse(ctx, requestID, botID, chatID, responseText); err != nil {
		return domain.WrapBusinessError(err)
	}
	return nil
}
