package kafka

import (
	"context"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/google/uuid"
)

// RAGRequestOptions опции для отправки RAG запроса
type RAGRequestOptions struct {
	// Headers опции
	Onboarding *bool // флаг onboarding
	Summarize  *bool // флаг summarize
	More       *bool // флаг more
	NeedPhoto  *bool // флаг need_photo
}

// IKafkaProducer интерфейс для отправки сообщений в Kafka
type IKafkaProducer interface {
	SendRAGRequest(ctx context.Context, requestID uuid.UUID, botID domain.BotId, chatID int64, requestText string, natalReport domain.NatalReport, requestType domain.RequestType) (partition int32, offset int64, err error)
	SendRAGRequestWithOptions(ctx context.Context, requestID uuid.UUID, botID domain.BotId, chatID int64, requestText string, natalReport domain.NatalReport, requestType domain.RequestType, options *RAGRequestOptions) (partition int32, offset int64, err error)
	SendRerankNatal(ctx context.Context, key string, botID domain.BotId, chatID int64, natalReport []byte) error
	Send(ctx context.Context, key string, value []byte) error
	Close() error
}
