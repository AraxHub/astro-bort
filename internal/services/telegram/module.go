package telegram

import (
	TgClient "github.com/admin/tg-bots/astro-bot/internal/adapters/secondary/telegram"
	"log/slog"

	"github.com/admin/tg-bots/astro-bot/internal/ports/service"
)

type Service struct {
	BotServices map[string]service.IBotService
	TgClient    *TgClient.Client
	Log         *slog.Logger
}

func New(
	botServices map[string]service.IBotService,
	tgClient *TgClient.Client,
	log *slog.Logger,
) *Service {
	return &Service{
		BotServices: botServices,
		TgClient:    tgClient,
		Log:         log,
	}
}
