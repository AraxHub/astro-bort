package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/admin/tg-bots/astro-bot/internal/app"
)

const appName = "astro_bot"

func main() {
	cfg, err := app.NewEnvConfig(appName)
	if err != nil {
		panic(err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	app := app.New(appName, cfg)

	err1 := app.Run(ctx)
	if err1 != nil {
		panic(err1)
	}
}
