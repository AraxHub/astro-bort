package app

import (
	"context"
	"fmt"

	"log/slog"

	"github.com/admin/tg-bots/astro-bot/internal/pkg/logger"
)

type App struct {
	Name string
	Cfg  *Config
	Log  *slog.Logger
}

func New(name string, cfg *Config) *App {
	return &App{
		Name: name,
		Cfg:  cfg,
		Log:  logger.New(name, cfg.Log),
	}
}

func (a *App) Run(ctx context.Context) error {
	a.Log.Info("starting application")

	deps, err := a.initDependencies(ctx)
	if err != nil {
		return fmt.Errorf("failed to init dependencies: %w", err)
	}

	return a.runServices(ctx, deps)
}
