package kafka

import (
	"context"

	"github.com/IBM/sarama"
)

// MessageHandler интерфейс для обработки сообщений из Kafka
type MessageHandler interface {
	HandleMessage(ctx context.Context, key string, value []byte, headers []sarama.RecordHeader) error
}
