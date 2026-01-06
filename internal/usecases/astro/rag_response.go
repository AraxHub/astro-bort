package astro

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/google/uuid"
)

// HandleRAGResponse обрабатывает ответ от RAG
func (s *Service) HandleRAGResponse(ctx context.Context, requestID uuid.UUID, responseText string) (err error) {
	var statusStage domain.RequestStage
	var statusErrorCode string
	var statusMetadata json.RawMessage
	var botID domain.BotId

	defer func() {
		if err != nil {
			// Ошибка - создаём статус ошибки
			errMsg := err.Error()
			metadata := domain.BuildErrorMetadata(
				statusStage,
				statusErrorCode,
				string(botID),
				map[string]interface{}{
					"request_id": requestID.String(),
				},
			)

			s.createOrLogStatus(ctx, &domain.Status{
				ID:           uuid.New(),
				ObjectType:   domain.ObjectTypeRequest,
				ObjectID:     requestID,
				Status:       domain.RequestError,
				ErrorMessage: &errMsg,
				Metadata:     metadata,
				CreatedAt:    time.Now(),
			})
		} else {
			// Успех - создаём финальный статус
			if statusMetadata == nil {
				// Если metadata не был установлен, не создаём статус
				return
			}

			s.createOrLogStatus(ctx, &domain.Status{
				ID:         uuid.New(),
				ObjectType: domain.ObjectTypeRequest,
				ObjectID:   requestID,
				Status:     domain.RequestCompleted,
				Metadata:   statusMetadata,
				CreatedAt:  time.Now(),
			})
		}
	}()

	// Получаем запрос
	req, err := s.RequestRepo.GetByID(ctx, requestID)
	if err != nil {
		statusStage = domain.StageGetRequest
		statusErrorCode = "DB_QUERY_ERROR"
		s.Log.Error("failed to get request by id",
			"error", err,
			"request_id", requestID,
		)
		return fmt.Errorf("failed to get request: %w", err)
	}

	botID = req.BotID

	// Сохраняем ответ
	req.ResponseText = responseText
	if err = s.RequestRepo.UpdateResponseText(ctx, req); err != nil {
		statusStage = domain.StageSaveResponse
		statusErrorCode = "DB_UPDATE_ERROR"
		s.Log.Error("failed to update request with response",
			"error", err,
			"request_id", requestID,
		)
		return fmt.Errorf("failed to update request: %w", err)
	}

	// Получаем пользователя
	user, err := s.UserRepo.GetByID(ctx, req.UserID)
	if err != nil {
		statusStage = domain.StageGetUser
		statusErrorCode = "DB_QUERY_ERROR"
		s.Log.Error("failed to get user by id",
			"error", err,
			"user_id", req.UserID,
			"request_id", requestID,
		)
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Отправляем сообщение
	messageID, err := s.TelegramService.SendMessageWithID(ctx, req.BotID, user.TelegramChatID, responseText)
	if err != nil {
		statusStage = domain.StageSendTelegram
		statusErrorCode = "TELEGRAM_SEND_ERROR"
		if strings.Contains(err.Error(), "429") {
			statusErrorCode = "TELEGRAM_RATE_LIMIT"
		} else if strings.Contains(err.Error(), "timeout") {
			statusErrorCode = "TELEGRAM_TIMEOUT"
		}
		s.Log.Error("failed to send RAG response to user",
			"error", err,
			"request_id", requestID,
			"user_id", user.ID,
			"bot_id", req.BotID,
		)
		return fmt.Errorf("failed to send response: %w", err)
	}

	// Успех - формируем metadata
	statusMetadata = domain.BuildTelegramMetadata(
		messageID,
		user.TelegramChatID,
		string(req.BotID),
		len(responseText),
	)

	s.Log.Info("RAG response sent to user",
		"request_id", requestID,
		"message_id", messageID,
	)

	return nil
}
