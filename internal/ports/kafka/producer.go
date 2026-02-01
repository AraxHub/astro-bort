package kafka

import (
	"context"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/google/uuid"
)

// IKafkaProducer интерфейс для отправки сообщений в Kafka
type IKafkaProducer interface {
	SendRAGRequest(ctx context.Context, requestID uuid.UUID, botID domain.BotId, chatID int64, requestText string, natalReport domain.NatalReport, requestType domain.RequestType) (partition int32, offset int64, err error)
	SendRerankNatal(ctx context.Context, key string, botID domain.BotId, chatID int64, natalReport []byte) error
	Send(ctx context.Context, key string, value []byte) error
	Close() error
}
