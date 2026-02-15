package astro

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	kafkaPorts "github.com/admin/tg-bots/astro-bot/internal/ports/kafka"
	"github.com/admin/tg-bots/astro-bot/internal/usecases/astro/texts"
	"github.com/google/uuid"
)

// HandleRAGCallback обрабатывает callback от кнопок ответов после онбординга
func (s *Service) HandleRAGCallback(ctx context.Context, botID domain.BotId, user *domain.User, callbackData string, messageID int64, chatID int64) error {
	// Парсим тип callback из data (формат: "button_{type}:{user_id}")
	parts := strings.Split(callbackData, ":")
	if len(parts) != 2 {
		s.Log.Warn("invalid button callback data format", "data", callbackData)
		return fmt.Errorf("invalid callback data format")
	}

	callbackType := parts[0] // button_summarize, button_more, button_special, button_special_back
	callbackUserID, err := uuid.Parse(parts[1])
	if err != nil {
		s.Log.Warn("failed to parse user_id from callback data", "error", err, "data", callbackData)
		return fmt.Errorf("failed to parse user_id: %w", err)
	}

	// Проверяем, что callback от того же пользователя
	if user.ID != callbackUserID {
		s.Log.Warn("callback user mismatch",
			"callback_user_id", callbackUserID,
			"actual_user_id", user.ID)
		return fmt.Errorf("callback user mismatch")
	}

	switch callbackType {
	case "button_summarize":
		return s.handleRAGSummarizeCallback(ctx, botID, user, messageID, chatID)
	case "button_more":
		return s.handleRAGMoreCallback(ctx, botID, user, messageID, chatID)
	case "button_special":
		return s.handleRAGSpecialCallback(ctx, botID, user, messageID, chatID)
	case "button_special_back":
		return s.handleRAGSpecialBackCallback(ctx, botID, user, messageID, chatID)
	default:
		s.Log.Warn("unknown button callback type", "type", callbackType)
		return fmt.Errorf("unknown callback type: %s", callbackType)
	}
}

// handleRAGSummarizeCallback обрабатывает нажатие на "Расскажи обо мне"
func (s *Service) handleRAGSummarizeCallback(ctx context.Context, botID domain.BotId, user *domain.User, messageID int64, chatID int64) error {
	s.Log.Info("handling RAG summarize callback",
		"user_id", user.ID,
		"chat_id", chatID,
	)

	// Создаём запрос в БД
	request := &domain.Request{
		ID:          uuid.New(),
		UserID:      user.ID,
		BotID:       botID,
		RequestType: domain.RequestTypeUser,
		RequestText: "", // Пустой текст для summarize
		CreatedAt:   time.Now(),
	}

	if err := s.RequestRepo.Create(ctx, request); err != nil {
		s.Log.Error("failed to create request for summarize",
			"error", err,
			"user_id", user.ID,
		)
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Получаем натальную карту
	natalReport, err := s.UserRepo.GetNatalChart(ctx, user.ID)
	if err != nil {
		s.Log.Error("failed to get natal chart for summarize",
			"error", err,
			"user_id", user.ID,
		)
		return fmt.Errorf("failed to get natal chart: %w", err)
	}

	// Отправляем техническое сообщение "Обрабатываю..."
	techMsgID, err := s.TelegramService.SendMessageWithID(ctx, botID, chatID, texts.UserQuestionReceived)
	if err != nil {
		s.Log.Error("failed to send tech message for summarize",
			"error", err,
			"request_id", request.ID,
			"chat_id", chatID,
		)
		// Продолжаем выполнение, даже если не удалось отправить сообщение
	} else {
		// Сохраняем информацию о запросе для последующего удаления технического сообщения
		s.setRequestInfo(chatID, request.ID, techMsgID)
	}

	// Сохраняем request_id как последний для этого чата
	s.setLastRequestID(chatID, request.ID)

	// Отправляем в Kafka с флагами: summarize=true, more=false, need_photo=true
	summarizeTrue := true
	moreFalse := false
	needPhotoTrue := true
	onboardingFalse := false

	options := &kafkaPorts.RAGRequestOptions{
		Onboarding: &onboardingFalse,
		Summarize:  &summarizeTrue,
		More:       &moreFalse,
		NeedPhoto:  &needPhotoTrue,
	}

	if s.KafkaProducer != nil {
		_, _, err := s.KafkaProducer.SendRAGRequestWithOptions(ctx, request.ID, botID, chatID, "", natalReport, domain.RequestTypeUser, options)
		if err != nil {
			s.Log.Error("failed to send summarize request to kafka",
				"error", err,
				"request_id", request.ID,
				"user_id", user.ID,
			)
			return fmt.Errorf("failed to send request to kafka: %w", err)
		}

		// Инкрементируем free_msg_count для бесплатных пользователей
		isPaidUser := user.IsPaid || user.ManualGranted
		if !isPaidUser {
			if err = s.UserRepo.UpdateFreeMsgCount(ctx, user.ID); err != nil {
				s.Log.Error("failed to increment free_msg_count for summarize",
					"error", err,
					"user_id", user.ID,
					"request_id", request.ID,
				)
				// Не возвращаем ошибку - сообщение уже отправлено в Kafka
			}
		}

		s.Log.Info("summarize request sent to kafka",
			"request_id", request.ID,
			"user_id", user.ID,
		)
	}

	return nil
}

// handleRAGMoreCallback обрабатывает нажатие на "Раскрой тему глубже"
func (s *Service) handleRAGMoreCallback(ctx context.Context, botID domain.BotId, user *domain.User, messageID int64, chatID int64) error {
	s.Log.Info("handling RAG more callback",
		"user_id", user.ID,
		"chat_id", chatID,
	)

	// Создаём запрос в БД
	request := &domain.Request{
		ID:          uuid.New(),
		UserID:      user.ID,
		BotID:       botID,
		RequestType: domain.RequestTypeUser,
		RequestText: "", // Пустой текст для more
		CreatedAt:   time.Now(),
	}

	if err := s.RequestRepo.Create(ctx, request); err != nil {
		s.Log.Error("failed to create request for more",
			"error", err,
			"user_id", user.ID,
		)
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Получаем натальную карту
	natalReport, err := s.UserRepo.GetNatalChart(ctx, user.ID)
	if err != nil {
		s.Log.Error("failed to get natal chart for more",
			"error", err,
			"user_id", user.ID,
		)
		return fmt.Errorf("failed to get natal chart: %w", err)
	}

	// Отправляем техническое сообщение "Обрабатываю..."
	techMsgID, err := s.TelegramService.SendMessageWithID(ctx, botID, chatID, texts.UserQuestionReceived)
	if err != nil {
		s.Log.Error("failed to send tech message for more",
			"error", err,
			"request_id", request.ID,
			"chat_id", chatID,
		)
		// Продолжаем выполнение, даже если не удалось отправить сообщение
	} else {
		// Сохраняем информацию о запросе для последующего удаления технического сообщения
		s.setRequestInfo(chatID, request.ID, techMsgID)
	}

	// Сохраняем request_id как последний для этого чата
	s.setLastRequestID(chatID, request.ID)

	// Отправляем в Kafka с флагами: summarize=false, more=true, need_photo=true
	summarizeFalse := false
	moreTrue := true
	needPhotoTrue := true
	onboardingFalse := false

	options := &kafkaPorts.RAGRequestOptions{
		Onboarding: &onboardingFalse,
		Summarize:  &summarizeFalse,
		More:       &moreTrue,
		NeedPhoto:  &needPhotoTrue,
	}

	if s.KafkaProducer != nil {
		_, _, err := s.KafkaProducer.SendRAGRequestWithOptions(ctx, request.ID, botID, chatID, "", natalReport, domain.RequestTypeUser, options)
		if err != nil {
			s.Log.Error("failed to send more request to kafka",
				"error", err,
				"request_id", request.ID,
				"user_id", user.ID,
			)
			return fmt.Errorf("failed to send request to kafka: %w", err)
		}

		// Инкрементируем free_msg_count для бесплатных пользователей
		isPaidUser := user.IsPaid || user.ManualGranted
		if !isPaidUser {
			if err = s.UserRepo.UpdateFreeMsgCount(ctx, user.ID); err != nil {
				s.Log.Error("failed to increment free_msg_count for more",
					"error", err,
					"user_id", user.ID,
					"request_id", request.ID,
				)
				// Не возвращаем ошибку - сообщение уже отправлено в Kafka
			}
		}

		s.Log.Info("more request sent to kafka",
			"request_id", request.ID,
			"user_id", user.ID,
		)
	}

	return nil
}

// handleRAGSpecialCallback обрабатывает нажатие на "Особые возможности"
func (s *Service) handleRAGSpecialCallback(ctx context.Context, botID domain.BotId, user *domain.User, messageID int64, chatID int64) error {
	s.Log.Info("handling RAG special callback",
		"user_id", user.ID,
		"chat_id", chatID,
	)

	// Создаём пустое меню с кнопкой "Назад"
	keyboard := map[string]interface{}{
		"inline_keyboard": [][]map[string]interface{}{
			{
				{
					"text":          "Назад",
					"callback_data": fmt.Sprintf("button_special_back:%s", user.ID.String()),
				},
			},
		},
	}

	// Редактируем сообщение, заменяя клавиатуру
	if err := s.TelegramService.EditMessageReplyMarkup(ctx, botID, chatID, messageID, keyboard); err != nil {
		s.Log.Error("failed to edit message reply markup",
			"error", err,
			"message_id", messageID,
			"chat_id", chatID,
		)
		return fmt.Errorf("failed to edit message: %w", err)
	}

	return nil
}

// handleRAGSpecialBackCallback обрабатывает нажатие на "Назад" из меню особых возможностей
func (s *Service) handleRAGSpecialBackCallback(ctx context.Context, botID domain.BotId, user *domain.User, messageID int64, chatID int64) error {
	s.Log.Info("handling RAG special back callback",
		"user_id", user.ID,
		"chat_id", chatID,
	)

	// Возвращаем основные 3 кнопки
	keyboard := s.buildRAGResponseKeyboard(user.ID)

	// Редактируем сообщение, заменяя клавиатуру
	if err := s.TelegramService.EditMessageReplyMarkup(ctx, botID, chatID, messageID, keyboard); err != nil {
		s.Log.Error("failed to edit message reply markup",
			"error", err,
			"message_id", messageID,
			"chat_id", chatID,
		)
		return fmt.Errorf("failed to edit message: %w", err)
	}

	return nil
}
