package astro

import (
	"log/slog"

	"github.com/admin/tg-bots/astro-bot/internal/ports/repository"
	"github.com/admin/tg-bots/astro-bot/internal/ports/telegram"
)

// Service бизнес-логика астро-бота
type Service struct {
	UserRepo       repository.IUserRepo
	RequestRepo    repository.IRequestRepo
	StatusRepo     repository.IStatusRepo
	TelegramClient telegram.IClient
	Log            *slog.Logger
}

// New создаёт новый сервис для бизнес-логики астро-бота
func New(
	userRepo repository.IUserRepo,
	requestRepo repository.IRequestRepo,
	statusRepo repository.IStatusRepo,
	telegramClient telegram.IClient,
	log *slog.Logger,
) *Service {
	return &Service{
		UserRepo:       userRepo,
		RequestRepo:    requestRepo,
		StatusRepo:     statusRepo,
		TelegramClient: telegramClient,
		Log:            log,
	}
}
