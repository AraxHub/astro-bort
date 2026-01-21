package astro

import (
	"context"
	"log/slog"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/admin/tg-bots/astro-bot/internal/ports/cache"
	"github.com/admin/tg-bots/astro-bot/internal/ports/kafka"
	"github.com/admin/tg-bots/astro-bot/internal/ports/repository"
	"github.com/admin/tg-bots/astro-bot/internal/ports/service"
	"github.com/admin/tg-bots/astro-bot/internal/ports/usecase"
)

// Service бизнес-логика астро-бота
type Service struct {
	UserRepo          repository.IUserRepo
	RequestRepo       repository.IRequestRepo
	StatusRepo        repository.IStatusRepo
	TelegramService   service.ITelegramService
	AstroAPIService   service.IAstroAPIService
	KafkaProducer     kafka.IKafkaProducer
	AlerterService    service.IAlerterService
	PaymentService    usecase.IPaymentUseCase // опциональный, для платежей
	PaymentRepo       repository.IPaymentRepo // опциональный, для получения данных о платежах
	Cache             cache.Cache
	FreeMessagesLimit int   // лимит бесплатных сообщений
	StarsPrice        int64 // цена подписки в звёздах
	Log               *slog.Logger
}

func New(
	userRepo repository.IUserRepo,
	requestRepo repository.IRequestRepo,
	statusRepo repository.IStatusRepo,
	telegramService service.ITelegramService,
	astroAPIService service.IAstroAPIService,
	kafkaProducer kafka.IKafkaProducer,
	alerterService service.IAlerterService,
	cache cache.Cache,
	freeMessagesLimit int,
	starsPrice int64,
	log *slog.Logger,
) *Service {
	return &Service{
		UserRepo:          userRepo,
		RequestRepo:       requestRepo,
		StatusRepo:        statusRepo,
		TelegramService:   telegramService,
		AstroAPIService:   astroAPIService,
		KafkaProducer:     kafkaProducer,
		AlerterService:    alerterService,
		PaymentService:    nil, // будет установлен через SetPaymentService
		PaymentRepo:       nil, // будет установлен через SetPaymentRepo
		Cache:             cache,
		FreeMessagesLimit: freeMessagesLimit,
		StarsPrice:        starsPrice,
		Log:               log,
	}
}

// SetPaymentService устанавливает payment service (опционально)
func (s *Service) SetPaymentService(paymentService usecase.IPaymentUseCase) {
	s.PaymentService = paymentService
}

// SetPaymentRepo устанавливает payment repo (опционально)
func (s *Service) SetPaymentRepo(paymentRepo repository.IPaymentRepo) {
	s.PaymentRepo = paymentRepo
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
