package kafka

import (
	"context"
	"fmt"

	"log/slog"

	"github.com/IBM/sarama"

	kafkaAdapter "github.com/admin/tg-bots/astro-bot/internal/adapters/secondary/kafka"
	"github.com/admin/tg-bots/astro-bot/internal/domain"
	kafkaPorts "github.com/admin/tg-bots/astro-bot/internal/ports/kafka"
)

// Consumer реализация Kafka consumer
type Consumer struct {
	consumer sarama.ConsumerGroup
	cfg      *kafkaAdapter.Config
	handler  kafkaPorts.MessageHandler
	log      *slog.Logger
}

// NewConsumer создаёт новый Kafka consumer
func NewConsumer(cfg *kafkaAdapter.Config, handler kafkaPorts.MessageHandler, log *slog.Logger) (*Consumer, error) {
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
	config.Consumer.Offsets.Initial = sarama.OffsetOldest

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

	consumer, err := sarama.NewConsumerGroup(cfg.GetBrokers(), cfg.ConsumerGroup, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka consumer: %w", err)
	}

	log.Info("kafka consumer created",
		"brokers", cfg.Brokers,
		"topic", cfg.Topic,
		"consumer_group", cfg.ConsumerGroup,
	)

	return &Consumer{
		consumer: consumer,
		cfg:      cfg,
		handler:  handler,
		log:      log,
	}, nil
}

// Start запускает consumer
func (c *Consumer) Start(ctx context.Context) error {
	handler := &consumerGroupHandler{
		handler: c.handler,
		log:     c.log,
		topic:   c.cfg.Topic,
	}

	for {
		select {
		case <-ctx.Done():
			c.log.Info("kafka consumer stopping", "topic", c.cfg.Topic)
			return c.consumer.Close()
		default:
			topics := []string{c.cfg.Topic}
			if err := c.consumer.Consume(ctx, topics, handler); err != nil {
				c.log.Error("error from consumer",
					"error", err,
					"topic", c.cfg.Topic,
				)
				return fmt.Errorf("consumer error: %w", err)
			}
		}
	}
}

// Close закрывает consumer
func (c *Consumer) Close() error {
	if err := c.consumer.Close(); err != nil {
		return fmt.Errorf("failed to close kafka consumer: %w", err)
	}
	c.log.Info("kafka consumer closed", "topic", c.cfg.Topic)
	return nil
}

// consumerGroupHandler реализует sarama.ConsumerGroupHandler
type consumerGroupHandler struct {
	handler kafkaPorts.MessageHandler
	log     *slog.Logger
	topic   string
}

func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	h.log.Info("kafka consumer group session setup", "topic", h.topic)
	return nil
}

func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	h.log.Info("kafka consumer group session cleanup", "topic", h.topic)
	return nil
}

// ConsumeClaim обрабатывает сообщения из Kafka
func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case <-session.Context().Done():
			return nil
		case message := <-claim.Messages():
			if message == nil {
				continue
			}

			key := string(message.Key)
			value := message.Value
			headers := make([]sarama.RecordHeader, len(message.Headers))
			for i, h := range message.Headers {
				headers[i] = *h
			}

			if err := h.handler.HandleMessage(session.Context(), key, value, headers); err != nil {
				if domain.IsBusinessError(err) {
					continue
				}
				h.log.Error("failed to handle kafka message",
					"error", err,
					"topic", message.Topic,
					"key", key,
					"partition", message.Partition,
					"offset", message.Offset,
				)

				//todo DLQ

				continue
			}

			// commit offset
			session.MarkMessage(message, "")
		}
	}
}
