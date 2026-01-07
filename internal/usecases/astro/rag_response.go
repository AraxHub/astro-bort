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
func (s *Service) HandleRAGResponse(ctx context.Context, requestID uuid.UUID, botID domain.BotId, chatID int64, responseText string) (err error) {
	var statusStage domain.RequestStage
	var statusErrorCode string
	var statusMetadata json.RawMessage

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
					"chat_id":    chatID,
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

	// Сохраняем ответ напрямую по request_id (без SELECT)
	if err = s.RequestRepo.UpdateResponseTextByID(ctx, requestID, responseText); err != nil {
		statusStage = domain.StageSaveResponse
		statusErrorCode = "DB_UPDATE_ERROR"
		s.Log.Error("failed to update request with response",
			"error", err,
			"request_id", requestID,
		)
		return fmt.Errorf("failed to update request: %w", err)
	}

	// Отправляем сообщение с bot_id и chat_id из Kafka (без SELECT User)
	messageID, err := s.TelegramService.SendMessageWithID(ctx, botID, chatID, responseText)
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
			"bot_id", botID,
			"chat_id", chatID,
		)
		return fmt.Errorf("failed to send response: %w", err)
	}

	// Успех - формируем metadata
	statusMetadata = domain.BuildTelegramMetadata(
		messageID,
		chatID,
		string(botID),
		len(responseText),
	)

	s.Log.Info("RAG response sent to user",
		"request_id", requestID,
		"message_id", messageID,
		"bot_id", botID,
		"chat_id", chatID,
	)

	return nil
}
