package app

import (
	server "github.com/admin/tg-bots/astro-bot/internal/adapters/primary/http"
	"github.com/admin/tg-bots/astro-bot/internal/adapters/secondary/storage/pg"
	"github.com/admin/tg-bots/astro-bot/internal/pkg/logger"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Postgres *pg.Config     `envconfig:"POSTGRES"`
	Log      *logger.Config `envconfig:"LOG"`
	Server   *server.Config `envconfig:"APISERVER"`
}

func NewEnvConfig(envPrefix string) (*Config, error) {
	cfg := &Config{}

	_ = godotenv.Load("deployments/local/.env")

	if err := envconfig.Process(envPrefix, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
