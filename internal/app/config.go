package app

import (
	"fmt"

	server "github.com/admin/tg-bots/astro-bot/internal/adapters/primary/http"
	alerterAdapter "github.com/admin/tg-bots/astro-bot/internal/adapters/secondary/alerter"
	astroApi "github.com/admin/tg-bots/astro-bot/internal/adapters/secondary/astroApi"
	kafkaAdapter "github.com/admin/tg-bots/astro-bot/internal/adapters/secondary/kafka"
	"github.com/admin/tg-bots/astro-bot/internal/adapters/secondary/storage/pg"
	"github.com/admin/tg-bots/astro-bot/internal/adapters/secondary/telegram"
	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/admin/tg-bots/astro-bot/internal/pkg/logger"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Postgres *pg.Config                `envconfig:"POSTGRES"`
	Log      *logger.Config            `envconfig:"LOG"`
	Server   *server.Config            `envconfig:"APISERVER"`
	Telegram *telegram.Config          `envconfig:"TELEGRAM"`
	AstroAPI *astroApi.Config          `envconfig:"ASTRO_API"`
	Bots     BotsConfig                `envconfig:"BOTS"`
	Kafka    kafkaAdapter.KafkaConfigs `envconfig:"KAFKA"`
	Alerter  *alerterAdapter.Config    `envconfig:"ALERTER"`
}

// BotsConfig конфигурация ботов
type BotsConfig struct {
	Count int         `envconfig:"COUNT" default:"1"`
	List  []BotConfig `envconfig:"-"` // Игнорируем envconfig, загружаем вручную
}

// Load загружает конфигурацию ботов из переменных окружения
func (bc *BotsConfig) Load(envPrefix string) error {
	bc.List = make([]BotConfig, bc.Count)
	for i := 0; i < bc.Count; i++ {
		prefix := fmt.Sprintf("%s_BOTS_%d", envPrefix, i) // TG_BOTS_BOTS_0, TG_BOTS_BOTS_1, ...
		var bot BotConfig
		if err := envconfig.Process(prefix, &bot); err != nil {
			return fmt.Errorf("failed to load bot %d: %w", i, err)
		}
		bc.List[i] = bot
	}
	return nil
}

func NewEnvConfig(envPrefix string) (*Config, error) {
	cfg := &Config{}

	_ = godotenv.Load("deployments/local/.env")

	if err := envconfig.Process(envPrefix, cfg); err != nil {
		return nil, err
	}

	// Загружаем ботов вручную (envconfig не умеет автоматически определять размер слайса)
	if err := cfg.Bots.Load(envPrefix); err != nil {
		return nil, fmt.Errorf("failed to load bots config: %w", err)
	}

	// Загружаем Kafka конфигурацию вручную
	if err := cfg.Kafka.Load(envPrefix); err != nil {
		return nil, fmt.Errorf("failed to load kafka config: %w", err)
	}

	return cfg, nil
}

// BotConfig конфигурация одного бота
type BotConfig struct {
	BotID    string `envconfig:"ID" required:"true"`    // TG_BOTS_BOTS_0_ID, TG_BOTS_BOTS_1_ID, ...
	BotType  string `envconfig:"TYPE" required:"true"`  // TG_BOTS_BOTS_0_TYPE, TG_BOTS_BOTS_1_TYPE, ...
	BotToken string `envconfig:"TOKEN" required:"true"` // TG_BOTS_BOTS_0_TOKEN, TG_BOTS_BOTS_1_TOKEN, ...
}

func (c *BotConfig) Validate() error {
	if c.BotID == "" {
		return fmt.Errorf("bot_id is required")
	}
	if c.BotType == "" {
		return fmt.Errorf("bot_type is required")
	}
	if c.BotToken == "" {
		return fmt.Errorf("bot_token is required")
	}

	// Проверяем валидность bot_type
	botType := domain.BotType(c.BotType)
	if !botType.IsValid() {
		return fmt.Errorf("invalid bot_type: %s", c.BotType)
	}

	return nil
}

func (c *BotConfig) ToDomain() (domain.BotId, domain.BotType, error) {
	if err := c.Validate(); err != nil {
		return "", "", fmt.Errorf("invalid bot config: %w", err)
	}

	return domain.BotId(c.BotID), domain.BotType(c.BotType), nil
}
