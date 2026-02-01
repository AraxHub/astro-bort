package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"log/slog"

	"github.com/IBM/sarama"
	"github.com/admin/tg-bots/astro-bot/internal/domain"
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
	if cfg.SecurityProtocol == "SASL_SSL" || cfg.SecurityProtocol == "SASL_PLAINTEXT" {
		config.Net.SASL.Enable = true
		config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		if cfg.SASLMechanism == "SCRAM-SHA-256" {
			config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
		}
		config.Net.SASL.User = cfg.SASLUsername
		config.Net.SASL.Password = cfg.SASLPassword
		// TLS только для SASL_SSL
		if cfg.SecurityProtocol == "SASL_SSL" {
			config.Net.TLS.Enable = true
		}
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

// SendRAGRequest отправляет запрос в RAG и возвращает partition и offset
// В value передаётся request_text и натальный отчёт (raw JSON без экранирования), остальные поля - в headers
func (p *Producer) SendRAGRequest(ctx context.Context, requestID uuid.UUID, botID domain.BotId, chatID int64, requestText string, natalReport domain.NatalReport, requestType domain.RequestType) (int32, int64, error) {
	var natalReportRaw json.RawMessage
	if len(natalReport) > 0 {
		if !json.Valid(natalReport) {
			return 0, 0, fmt.Errorf("natal_report is not valid JSON")
		}
		natalReportRaw = json.RawMessage(natalReport)
	}

	valueData := map[string]interface{}{
		"request_text": requestText,
		"natal_chart":  natalReportRaw,
	}
	valueBytes, err := json.Marshal(valueData)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to marshal value: %w", err)
	}

	headers := []sarama.RecordHeader{
		{
			Key:   []byte("request_id"),
			Value: []byte(requestID.String()),
		},
		{
			Key:   []byte("bot_id"),
			Value: []byte(string(botID)),
		},
		{
			Key:   []byte("chat_id"),
			Value: []byte(fmt.Sprintf("%d", chatID)),
		},
	}

	if action := requestType.KafkaAction(); action != "" {
		headers = append(headers, sarama.RecordHeader{
			Key:   []byte("action"),
			Value: []byte(action),
		})
	}

	msg := &sarama.ProducerMessage{
		Topic:   p.cfg.Topic,
		Key:     sarama.StringEncoder(requestID.String()),
		Value:   sarama.ByteEncoder(valueBytes),
		Headers: headers,
	}

	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		p.log.Debug("kafka send failed",
			"error", err,
			"topic", p.cfg.Topic,
			"key", requestID.String(),
		)
		return 0, 0, fmt.Errorf("kafka send failed [topic=%s, key=%s]: %w",
			p.cfg.Topic, requestID.String(), err)
	}

	p.log.Debug("message sent to kafka",
		"topic", p.cfg.Topic,
		"partition", partition,
		"offset", offset,
		"key", requestID.String(),
		"chat_id", chatID,
	)

	return partition, offset, nil
}

// SendRerankNatal отправляет натальную карту для rerank с нужными headers
func (p *Producer) SendRerankNatal(ctx context.Context, key string, botID domain.BotId, chatID int64, natalReport []byte) error {
	headers := []sarama.RecordHeader{
		{
			Key:   []byte("action"),
			Value: []byte("rerank_natal"),
		},
		{
			Key:   []byte("bot_id"),
			Value: []byte(string(botID)),
		},
		{
			Key:   []byte("chat_id"),
			Value: []byte(fmt.Sprintf("%d", chatID)),
		},
	}

	msg := &sarama.ProducerMessage{
		Topic:   p.cfg.Topic,
		Key:     sarama.StringEncoder(key),
		Value:   sarama.ByteEncoder(natalReport),
		Headers: headers,
	}

	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		p.log.Debug("kafka send rerank natal failed",
			"error", err,
			"topic", p.cfg.Topic,
			"key", key,
			"bot_id", botID,
			"chat_id", chatID,
		)
		return fmt.Errorf("kafka send rerank natal failed [topic=%s, key=%s]: %w",
			p.cfg.Topic, key, err)
	}

	p.log.Debug("natal report sent to kafka for rerank",
		"topic", p.cfg.Topic,
		"partition", partition,
		"offset", offset,
		"key", key,
		"bot_id", botID,
		"chat_id", chatID,
	)

	return nil
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
		// Debug для технических деталей
		p.log.Debug("kafka send failed",
			"error", err,
			"topic", p.cfg.Topic,
			"key", key,
		)
		// Оборачиваем с техническими деталями
		return fmt.Errorf("kafka send failed [topic=%s, key=%s]: %w",
			p.cfg.Topic, key, err)
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
