package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"log/slog"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
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
		return fmt.Errorf("failed to unmarshal RAG response: %w", err)
	}

	requestID, err := uuid.Parse(response.RequestID)
	if err != nil {
		return fmt.Errorf("invalid request_id: %w", err)
	}

	botID := domain.BotId(response.BotID)
	if botID == "" {
		return fmt.Errorf("bot_id is required in RAG response")
	}

	if response.ChatID == 0 {
		return fmt.Errorf("chat_id is required in RAG response")
	}

	h.Log.Debug("processing RAG response",
		"request_id", requestID,
		"bot_id", botID,
		"chat_id", response.ChatID,
		"response_length", len(response.ResponseText),
	)

	if err := h.TelegramService.HandleRAGResponse(ctx, requestID, botID, response.ChatID, response.ResponseText); err != nil {
		return fmt.Errorf("failed to handle RAG response: %w", err)
	}

	return nil
}

// RAGResponseMessage структура ответа от RAG
type RAGResponseMessage struct {
	RequestID    string `json:"request_id"`
	BotID        string `json:"bot_id"`
	ChatID       int64  `json:"chat_id"`
	ResponseText string `json:"response_text"`
}
