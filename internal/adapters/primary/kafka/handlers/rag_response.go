package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"log/slog"

	"github.com/IBM/sarama"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	astroUsecase "github.com/admin/tg-bots/astro-bot/internal/usecases/astro"
	"github.com/google/uuid"

	kafkaPorts "github.com/admin/tg-bots/astro-bot/internal/ports/kafka"
	telegramService "github.com/admin/tg-bots/astro-bot/internal/services/telegram"
)

// RAGResponseHandler обрабатывает ответы от RAG
type RAGResponseHandler struct {
	AstroService    *astroUsecase.Service
	TelegramService *telegramService.Service
	Log             *slog.Logger
}

// NewRAGResponseHandler создаёт новый handler для ответов от RAG
func NewRAGResponseHandler(
	astroService *astroUsecase.Service,
	telegramService *telegramService.Service,
	log *slog.Logger,
) kafkaPorts.MessageHandler {
	return &RAGResponseHandler{
		AstroService:    astroService,
		TelegramService: telegramService,
		Log:             log,
	}
}

// getHeaderValue получает значение хэдера по ключу
func getHeaderValue(headers []sarama.RecordHeader, key string) string {
	for _, header := range headers {
		if string(header.Key) == key {
			return string(header.Value)
		}
	}
	return ""
}

// HandleMessage обрабатывает сообщение от RAG
func (h *RAGResponseHandler) HandleMessage(ctx context.Context, key string, value []byte, headers []sarama.RecordHeader) error {
	action := getHeaderValue(headers, "action")

	// Парсим value
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

	switch action {
	case "image_type":
		return h.handleImageResponse(ctx, requestID, botID, response.ChatID, response.ResponseText)
	case "Nothing":
		return nil
	case "chat":
		return h.handleTextResponse(ctx, requestID, botID, response.ChatID, response.ResponseText)
	default:
		return h.handleTextResponse(ctx, requestID, botID, response.ChatID, response.ResponseText)
	}
}

// handleImageResponse обрабатывает ответ с темой для фото
func (h *RAGResponseHandler) handleImageResponse(ctx context.Context, requestID uuid.UUID, botID domain.BotId, chatID int64, theme string) error {
	if h.AstroService != nil && !h.AstroService.IsLastRequestID(chatID, requestID) {
		h.Log.Debug("ignoring image response for outdated request",
			"request_id", requestID,
			"chat_id", chatID,
			"theme", theme)
		return nil // Не ошибка, просто устаревший запрос
	}

	if h.AstroService != nil {
		return h.AstroService.SendImageForTheme(ctx, requestID, botID, chatID, theme)
	}

	return fmt.Errorf("astro service not configured")
}

// handleTextResponse обрабатывает текстовый ответ
func (h *RAGResponseHandler) handleTextResponse(ctx context.Context, requestID uuid.UUID, botID domain.BotId, chatID int64, responseText string) error {
	return h.TelegramService.HandleRAGResponse(ctx, requestID, botID, chatID, responseText)
}

// RAGResponseMessage структура ответа от RAG
type RAGResponseMessage struct {
	RequestID    string `json:"request_id"`
	BotID        string `json:"bot_id"`
	ChatID       int64  `json:"chat_id"`
	ResponseText string `json:"response_text"`
}
