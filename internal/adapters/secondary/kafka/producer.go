package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"log/slog"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
)

// Producer реализация Kafka producer
type Producer struct {
	producer sarama.SyncProducer
	cfg      *Config
	log      *slog.Logger
}

// NewProducer создаёт новый Kafka producer
func NewProducer(cfg *Config, log *slog.Logger) (*Producer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5

	// Настройка безопасности (если указано)
	if cfg.SecurityProtocol == "SASL_SSL" {
		config.Net.SASL.Enable = true
		config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		if cfg.SASLMechanism == "SCRAM-SHA-256" {
			config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
		}
		config.Net.SASL.User = cfg.SASLUsername
		config.Net.SASL.Password = cfg.SASLPassword
		config.Net.TLS.Enable = true
	}

	producer, err := sarama.NewSyncProducer(cfg.GetBrokers(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka producer: %w", err)
	}

	log.Info("kafka producer created",
		"brokers", cfg.Brokers,
		"topic", cfg.Topic,
	)

	return &Producer{
		producer: producer,
		cfg:      cfg,
		log:      log,
	}, nil
}

// SendRAGRequest отправляет запрос в RAG
func (p *Producer) SendRAGRequest(ctx context.Context, requestID uuid.UUID, requestText string, natalChart []byte) error {
	message := RAGRequestMessage{
		RequestID:   requestID.String(),
		RequestText: requestText,
		NatalChart:  string(natalChart),
	}

	value, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal RAG request: %w", err)
	}

	return p.Send(ctx, requestID.String(), value)
}

// Send отправляет произвольное сообщение
func (p *Producer) Send(ctx context.Context, key string, value []byte) error {
	msg := &sarama.ProducerMessage{
		Topic: p.cfg.Topic,
		Key:   sarama.StringEncoder(key),
		Value: sarama.ByteEncoder(value),
	}

	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		p.log.Error("failed to send message to kafka",
			"error", err,
			"topic", p.cfg.Topic,
			"key", key,
		)
		return fmt.Errorf("failed to send message to kafka: %w", err)
	}

	p.log.Debug("message sent to kafka",
		"topic", p.cfg.Topic,
		"partition", partition,
		"offset", offset,
		"key", key,
	)

	return nil
}

// Close закрывает producer
func (p *Producer) Close() error {
	if err := p.producer.Close(); err != nil {
		return fmt.Errorf("failed to close kafka producer: %w", err)
	}
	p.log.Info("kafka producer closed")
	return nil
}

// RAGRequestMessage структура сообщения для RAG
type RAGRequestMessage struct {
	RequestID   string `json:"request_id"`
	RequestText string `json:"request_text"`
	NatalChart  string `json:"natal_chart"`
}
