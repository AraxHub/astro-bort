package kafka

import (
	"fmt"
	"strings"

	"github.com/kelseyhightower/envconfig"
)

// Config конфигурация для Kafka producer/consumer
type Config struct {
	Brokers          string `envconfig:"BROKERS"`           // "broker1:9092,broker2:9092"
	Topic            string `envconfig:"TOPIC"`             // название топика
	ConsumerGroup    string `envconfig:"CONSUMER_GROUP"`    // consumer group (только для consumer)
	SecurityProtocol string `envconfig:"SECURITY_PROTOCOL"` // "SASL_SSL", "PLAINTEXT"
	SASLMechanism    string `envconfig:"SASL_MECHANISM"`    // "PLAIN", "SCRAM-SHA-256"
	SASLUsername     string `envconfig:"SASL_USERNAME"`
	SASLPassword     string `envconfig:"SASL_PASSWORD"`
}

// GetBrokers возвращает список брокеров из строки
func (c *Config) GetBrokers() []string {
	if c.Brokers == "" {
		return []string{"localhost:9092"}
	}
	return strings.Split(c.Brokers, ",")
}

// KafkaConfigs конфигурация для нескольких Kafka кластеров/топиков
type KafkaConfigs struct {
	Count int           `envconfig:"COUNT" default:"1"`
	List  []KafkaConfig `envconfig:"-"`
}

// KafkaConfig конфигурация одного Kafka подключения
type KafkaConfig struct {
	Name   string  `envconfig:"NAME"` // "rag_requests", "rag_responses"
	Config *Config `envconfig:"CONFIG"`
}

// Load загружает конфигурацию Kafka из переменных окружения
func (kc *KafkaConfigs) Load(envPrefix string) error {
	kc.List = make([]KafkaConfig, kc.Count)
	for i := 0; i < kc.Count; i++ {
		prefix := fmt.Sprintf("%s_KAFKA_%d", envPrefix, i) // TG_BOTS_KAFKA_0, TG_BOTS_KAFKA_1, ...
		var kafkaCfg KafkaConfig
		if err := envconfig.Process(prefix, &kafkaCfg); err != nil {
			return fmt.Errorf("failed to load kafka config %d: %w", i, err)
		}
		kc.List[i] = kafkaCfg
	}
	return nil
}
