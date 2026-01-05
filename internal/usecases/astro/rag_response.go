package astro

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// HandleRAGResponse обрабатывает ответ от RAG
func (s *Service) HandleRAGResponse(ctx context.Context, requestID uuid.UUID, responseText string) error {
	// Получаем request по request_id
	request, err := s.RequestRepo.GetByID(ctx, requestID)
	if err != nil {
		s.Log.Error("failed to get request by id",
			"error", err,
			"request_id", requestID,
		)
		return fmt.Errorf("failed to get request: %w", err)
	}

	// Получаем user по user_id из request
	user, err := s.UserRepo.GetByID(ctx, request.UserID)
	if err != nil {
		s.Log.Error("failed to get user by id",
			"error", err,
			"user_id", request.UserID,
			"request_id", requestID,
		)
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Отправляем ответ пользователю через Telegram Service
	if err := s.sendMessage(ctx, request.BotID, user.TelegramChatID, responseText); err != nil {
		s.Log.Error("failed to send RAG response to user",
			"error", err,
			"request_id", requestID,
			"user_id", user.ID,
			"bot_id", request.BotID,
		)
		return fmt.Errorf("failed to send response: %w", err)
	}

	s.Log.Info("RAG response sent to user",
		"request_id", requestID,
		"user_id", user.ID,
		"bot_id", request.BotID,
		"response_length", len(responseText),
	)

	return nil
}
