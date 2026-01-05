package kafka

import (
	"context"

	"github.com/google/uuid"
)

// IKafkaProducer интерфейс для отправки сообщений в Kafka
type IKafkaProducer interface {
	// SendRAGRequest отправляет запрос в RAG
	SendRAGRequest(ctx context.Context, requestID uuid.UUID, requestText string, natalChart []byte) error
	// Send отправляет произвольное сообщение
	Send(ctx context.Context, key string, value []byte) error
	// Close закрывает producer
	Close() error
}
