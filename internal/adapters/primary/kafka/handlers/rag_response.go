package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"log/slog"

	"github.com/google/uuid"

	kafkaPorts "github.com/admin/tg-bots/astro-bot/internal/ports/kafka"
	telegramService "github.com/admin/tg-bots/astro-bot/internal/services/telegram"
)

// RAGResponseHandler обрабатывает ответы от RAG
type RAGResponseHandler struct {
	TelegramService *telegramService.Service
	Log             *slog.Logger
}

// NewRAGResponseHandler создаёт новый handler для ответов от RAG
func NewRAGResponseHandler(telegramService *telegramService.Service, log *slog.Logger) kafkaPorts.MessageHandler {
	return &RAGResponseHandler{
		TelegramService: telegramService,
		Log:             log,
	}
}

// HandleMessage обрабатывает сообщение от RAG
func (h *RAGResponseHandler) HandleMessage(ctx context.Context, key string, value []byte) error {
	var response RAGResponseMessage
	if err := json.Unmarshal(value, &response); err != nil {
		h.Log.Error("failed to unmarshal RAG response",
			"error", err,
			"key", key,
		)
		return fmt.Errorf("failed to unmarshal RAG response: %w", err)
	}

	requestID, err := uuid.Parse(response.RequestID)
	if err != nil {
		h.Log.Error("invalid request_id in RAG response",
			"error", err,
			"request_id", response.RequestID,
		)
		return fmt.Errorf("invalid request_id: %w", err)
	}

	h.Log.Info("processing RAG response",
		"request_id", requestID,
		"response_length", len(response.ResponseText),
	)

	// Вызываем Telegram Service для обработки ответа
	if err := h.TelegramService.HandleRAGResponse(ctx, requestID, response.ResponseText); err != nil {
		h.Log.Error("failed to handle RAG response",
			"error", err,
			"request_id", requestID,
		)
		return fmt.Errorf("failed to handle RAG response: %w", err)
	}

	return nil
}

// RAGResponseMessage структура ответа от RAG
type RAGResponseMessage struct {
	RequestID    string `json:"request_id"`
	ResponseText string `json:"response_text"`
}
