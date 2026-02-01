package astro

import (
	"context"
	"log/slog"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/admin/tg-bots/astro-bot/internal/ports/cache"
	"github.com/admin/tg-bots/astro-bot/internal/ports/kafka"
	"github.com/admin/tg-bots/astro-bot/internal/ports/repository"
	"github.com/admin/tg-bots/astro-bot/internal/ports/service"
	"github.com/admin/tg-bots/astro-bot/internal/ports/storage"
	"github.com/admin/tg-bots/astro-bot/internal/ports/usecase"
	"github.com/google/uuid"
)

// Service бизнес-логика астро-бота
type Service struct {
	UserRepo          repository.IUserRepo
	RequestRepo       repository.IRequestRepo
	StatusRepo        repository.IStatusRepo
	ImageRepo         repository.IImageRepo      // опциональный, для работы с картинками
	ImageUsageRepo    repository.IImageUsageRepo // опциональный, для статистики использования картинок
	TelegramService   service.ITelegramService
	AstroAPIService   service.IAstroAPIService
	KafkaProducer     kafka.IKafkaProducer
	AlerterService    service.IAlerterService
	PaymentService    usecase.IPaymentUseCase // опциональный, для платежей
	PaymentRepo       repository.IPaymentRepo // опциональный, для получения данных о платежах
	Cache             cache.Cache
	RequestCache      cache.IRequestCache // кэш последних request_id по chat_id
	S3Client          storage.IS3Client   // опциональный, для работы с картинками
	FreeMessagesLimit int                 // лимит бесплатных сообщений
	StarsPrice        int64               // цена подписки в звёздах
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
	requestCache cache.IRequestCache,
	s3Client storage.IS3Client,
	imageRepo repository.IImageRepo,
	imageUsageRepo repository.IImageUsageRepo,
	freeMessagesLimit int,
	starsPrice int64,
	log *slog.Logger,
) *Service {
	return &Service{
		UserRepo:          userRepo,
		RequestRepo:       requestRepo,
		StatusRepo:        statusRepo,
		ImageRepo:         imageRepo,      // может быть nil
		ImageUsageRepo:    imageUsageRepo, // может быть nil
		TelegramService:   telegramService,
		AstroAPIService:   astroAPIService,
		KafkaProducer:     kafkaProducer,
		AlerterService:    alerterService,
		PaymentService:    nil, // будет установлен через SetPaymentService
		PaymentRepo:       nil, // будет установлен через SetPaymentRepo
		Cache:             cache,
		RequestCache:      requestCache,
		S3Client:          s3Client, // может быть nil
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

// setLastRequestID сохраняет последний request_id для chat_id (при отправке в Kafka)
func (s *Service) setLastRequestID(chatID int64, requestID uuid.UUID) {
	if s.RequestCache != nil {
		s.RequestCache.SetLastRequestID(chatID, requestID)
	}
}

// IsLastRequestID публичный метод для проверки последнего request_id (для Kafka handler)
func (s *Service) IsLastRequestID(chatID int64, requestID uuid.UUID) bool {
	if s.RequestCache == nil {
		return true
	}
	return s.RequestCache.IsLastRequestID(chatID, requestID)
}
