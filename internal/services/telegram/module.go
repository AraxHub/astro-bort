package telegram

import (
	"fmt"

	"log/slog"

	TgClient "github.com/admin/tg-bots/astro-bot/internal/adapters/secondary/telegram"
	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/admin/tg-bots/astro-bot/internal/ports/repository"
	"github.com/admin/tg-bots/astro-bot/internal/ports/service"
)

type Service struct {
	BotIDToType      map[domain.BotId]domain.BotType        // botID → botType (для роутинга к UseCase)
	BotTypeToUsecase map[domain.BotType]service.IBotService // botType → UseCase
	TelegramClients  map[domain.BotId]*TgClient.Client      // botID → Client
	RequestRepo      repository.IRequestRepo                // для получения request по ID
	Log              *slog.Logger
}

func New(
	botIDToType map[domain.BotId]domain.BotType,
	botServices map[domain.BotType]service.IBotService,
	telegramClients map[domain.BotId]*TgClient.Client,
	requestRepo repository.IRequestRepo,
	log *slog.Logger,
) *Service {
	return &Service{
		BotIDToType:      botIDToType,
		BotTypeToUsecase: botServices,
		TelegramClients:  telegramClients,
		RequestRepo:      requestRepo,
		Log:              log,
	}
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

//todo подумать норм ли, что репозитории есть и в бизнес-логике и в телеграм сервисе
