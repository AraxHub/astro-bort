package app

import (
	starsProvider "github.com/admin/tg-bots/astro-bot/internal/adapters/secondary/payment/telegram_stars"
	tgAdapter "github.com/admin/tg-bots/astro-bot/internal/adapters/secondary/telegram"
	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/admin/tg-bots/astro-bot/internal/ports/service"
	telegramService "github.com/admin/tg-bots/astro-bot/internal/services/telegram"
	astroUsecase "github.com/admin/tg-bots/astro-bot/internal/usecases/astro"
	paymentUsecase "github.com/admin/tg-bots/astro-bot/internal/usecases/payment"
)

// initPayment инициализирует payment use case и интегрирует его в telegram service
// Возвращает созданный payment use case для использования в других use cases
func (a *App) initPayment(
	telegramClients map[domain.BotId]*tgAdapter.Client,
	repos *repositories,
	tgService *telegramService.Service,
	alerterSvc service.IAlerterService,
	astroUseCase *astroUsecase.Service,
) *paymentUsecase.Service {
	if len(telegramClients) == 0 {
		a.Log.Warn("no telegram clients available, payment system disabled")
		return nil
	}

	// Конвертируем map[domain.BotId]*Client в map[string]*Client для провайдера
	clientsMap := make(map[string]*tgAdapter.Client)
	for botID, client := range telegramClients {
		clientsMap[string(botID)] = client
	}

	// Создаём Telegram Stars провайдер с поддержкой нескольких ботов
	paymentProvider := starsProvider.NewProvider(clientsMap, a.Log)

	// Создаём payment use case
	paymentUseCase := paymentUsecase.New(
		repos.Payment,
		repos.User,
		repos.Status,
		paymentProvider,
		tgService,
		alerterSvc, // может быть nil
		a.Log,
	)

	// Интегрируем payment use case в telegram service
	tgService.SetPaymentUseCase(paymentUseCase)

	// Устанавливаем payment service и payment repo в astro use case
	if astroUseCase != nil {
		astroUseCase.SetPaymentService(paymentUseCase)
		astroUseCase.SetPaymentRepo(repos.Payment)
	}

	a.Log.Info("payment system initialized successfully")
	return paymentUseCase
}
