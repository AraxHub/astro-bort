package astro

import (
	"context"
	"log/slog"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/admin/tg-bots/astro-bot/internal/ports/kafka"
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
	KafkaProducer   kafka.IKafkaProducer
	Log             *slog.Logger
}

func New(
	userRepo repository.IUserRepo,
	requestRepo repository.IRequestRepo,
	statusRepo repository.IStatusRepo,
	telegramService service.ITelegramService,
	astroAPIService service.IAstroAPIService,
	kafkaProducer kafka.IKafkaProducer,
	log *slog.Logger,
) *Service {
	return &Service{
		UserRepo:        userRepo,
		RequestRepo:     requestRepo,
		StatusRepo:      statusRepo,
		TelegramService: telegramService,
		AstroAPIService: astroAPIService,
		KafkaProducer:   kafkaProducer,
		Log:             log,
	}
}

// createOrLogStatus не падает если БД недоступна
func (s *Service) createOrLogStatus(ctx context.Context, status *domain.Status) {
	if err := s.StatusRepo.Create(ctx, status); err != nil {
		s.Log.Warn("failed to create status (non-critical)",
			"error", err,
			"object_id", status.ObjectID,
			"status", status.Status,
		)
	}
}
