package kafka

import "context"

// MessageHandler интерфейс для обработки сообщений из Kafka
type MessageHandler interface {
	HandleMessage(ctx context.Context, key string, value []byte) error
}
