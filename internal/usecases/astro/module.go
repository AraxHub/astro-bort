package astro

import (
	"log/slog"

	"github.com/admin/tg-bots/astro-bot/internal/ports/repository"
	"github.com/admin/tg-bots/astro-bot/internal/ports/service"
)

// Service бизнес-логика астро-бота
type Service struct {
	UserRepo        repository.IUserRepo
	RequestRepo     repository.IRequestRepo
	StatusRepo      repository.IStatusRepo
	TelegramService service.ITelegramService
	AstroAPIService service.IAstroAPIService
	Log             *slog.Logger
}

// New создаёт новый сервис для бизнес-логики астро-бота
func New(
	userRepo repository.IUserRepo,
	requestRepo repository.IRequestRepo,
	statusRepo repository.IStatusRepo,
	telegramService service.ITelegramService,
	astroAPIService service.IAstroAPIService,
	log *slog.Logger,
) *Service {
	return &Service{
		UserRepo:        userRepo,
		RequestRepo:     requestRepo,
		StatusRepo:      statusRepo,
		TelegramService: telegramService,
		AstroAPIService: astroAPIService,
		Log:             log,
	}
}
