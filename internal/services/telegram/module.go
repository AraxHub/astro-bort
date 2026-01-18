package telegram

import (
	"context"
	"fmt"

	"log/slog"

	TgClient "github.com/admin/tg-bots/astro-bot/internal/adapters/secondary/telegram"
	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/admin/tg-bots/astro-bot/internal/ports/service"
	"github.com/google/uuid"
)

// PaymentUseCase интерфейс для работы с платежами (чтобы не создавать циклическую зависимость)
type PaymentUseCase interface {
	HandlePreCheckoutQuery(ctx context.Context, botID domain.BotId, queryID string, userID uuid.UUID, amount int64, currency string, payload string) (bool, error)
	HandleSuccessfulPayment(ctx context.Context, botID domain.BotId, userID uuid.UUID, chatID int64, paymentID uuid.UUID, telegramPaymentChargeID string) error
}

type Service struct {
	BotIDToType      map[domain.BotId]domain.BotType        // botID → botType (для роутинга к UseCase)
	BotTypeToUsecase map[domain.BotType]service.IBotService // botType → UseCase
	TelegramClients  map[domain.BotId]*TgClient.Client      // botID → Client
	PaymentUseCase   PaymentUseCase                         // use case для платежей (опционально)
	Log              *slog.Logger
}

func New(
	botIDToType map[domain.BotId]domain.BotType,
	botServices map[domain.BotType]service.IBotService,
	telegramClients map[domain.BotId]*TgClient.Client,
	log *slog.Logger,
) *Service {
	return &Service{
		BotIDToType:      botIDToType,
		BotTypeToUsecase: botServices,
		TelegramClients:  telegramClients,
		PaymentUseCase:   nil, // будет установлен через SetPaymentUseCase
		Log:              log,
	}
}

// SetPaymentUseCase устанавливает payment use case (для обработки платежей)
func (s *Service) SetPaymentUseCase(paymentUseCase PaymentUseCase) {
	s.PaymentUseCase = paymentUseCase
}

// SetBotServices устанавливает botServices (для случаев когда нужно обновить после создания)
func (s *Service) SetBotServices(botServices map[domain.BotType]service.IBotService) {
	s.BotTypeToUsecase = botServices
}

// GetBotType возвращает botType для указанного botID
func (s *Service) GetBotType(botID domain.BotId) (domain.BotType, error) {
	botType, ok := s.BotIDToType[botID]
	if !ok {
		return "", fmt.Errorf("bot_type not found for bot_id: %s", botID)
	}
	return botType, nil
}
