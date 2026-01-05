package app

import (
	"context"
	"fmt"
	"net/http"

	server "github.com/admin/tg-bots/astro-bot/internal/adapters/primary/http"
	healthcheckController "github.com/admin/tg-bots/astro-bot/internal/adapters/primary/http/controllers/healthcheck"
	telegramController "github.com/admin/tg-bots/astro-bot/internal/adapters/primary/http/controllers/telegram"
	testController "github.com/admin/tg-bots/astro-bot/internal/adapters/primary/http/controllers/test"
	kafkaConsumerAdapter "github.com/admin/tg-bots/astro-bot/internal/adapters/primary/kafka"
	kafkaHandlers "github.com/admin/tg-bots/astro-bot/internal/adapters/primary/kafka/handlers"
	astroApiAdapter "github.com/admin/tg-bots/astro-bot/internal/adapters/secondary/astroApi"
	kafkaAdapter "github.com/admin/tg-bots/astro-bot/internal/adapters/secondary/kafka"
	"github.com/admin/tg-bots/astro-bot/internal/adapters/secondary/storage/pg"
	tgAdapter "github.com/admin/tg-bots/astro-bot/internal/adapters/secondary/telegram"
	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/admin/tg-bots/astro-bot/internal/ports/kafka"
	"github.com/admin/tg-bots/astro-bot/internal/ports/service"
	requestRepo "github.com/admin/tg-bots/astro-bot/internal/repository/request"
	statusRepo "github.com/admin/tg-bots/astro-bot/internal/repository/status"
	testRepo "github.com/admin/tg-bots/astro-bot/internal/repository/test"
	userRepo "github.com/admin/tg-bots/astro-bot/internal/repository/user"
	astroApiService "github.com/admin/tg-bots/astro-bot/internal/services/astroApi"
	telegramService "github.com/admin/tg-bots/astro-bot/internal/services/telegram"
	astroUsecase "github.com/admin/tg-bots/astro-bot/internal/usecases/astro"
	testService "github.com/admin/tg-bots/astro-bot/internal/usecases/test"
	"github.com/jmoiron/sqlx"
)

type Dependencies struct {
	DB              *sqlx.DB
	HTTPServer      *http.Server
	TelegramService *telegramService.Service
	TelegramClients map[domain.BotId]*tgAdapter.Client
	TelegramPoller  *tgAdapter.Poller
	KafkaProducers  map[string]*kafkaAdapter.Producer
	KafkaConsumers  map[string]*kafkaConsumerAdapter.Consumer
}

func (a *App) initDependencies(ctx context.Context) (*Dependencies, error) {
	db, err := a.initPostgres()
	if err != nil {
		return nil, fmt.Errorf("failed to init postgres: %w", err)
	}

	persistenceLayer := pg.NewDB(db)

	userRepo := userRepo.New(persistenceLayer, a.Log)
	requestRepo := requestRepo.New(persistenceLayer, a.Log)
	statusRepo := statusRepo.New(persistenceLayer, a.Log)
	testRepo := testRepo.New(persistenceLayer, a.Log)

	if len(a.Cfg.Bots.List) == 0 {
		return nil, fmt.Errorf("no bots configured: at least one bot must be specified via BOTS_COUNT and BOTS_0_* environment variables")
	}

	botIDToType := make(map[domain.BotId]domain.BotType)
	telegramClients := make(map[domain.BotId]*tgAdapter.Client)

	for i, botCfg := range a.Cfg.Bots.List {
		botID, botType, err := botCfg.ToDomain()
		if err != nil {
			return nil, fmt.Errorf("failed to convert bot config at index %d: %w", i, err)
		}

		botIDToType[botID] = botType
		telegramClients[botID] = tgAdapter.NewClient(botCfg.BotToken, a.Log)

		// Регистрируем команды для каждого бота
		if err := a.registerBotCommands(ctx, telegramClients[botID]); err != nil {
			a.Log.Warn("failed to register bot commands",
				"error", err,
				"bot_id", botID,
			)
		}
	}

	tgService := telegramService.New(
		botIDToType,
		make(map[domain.BotType]service.IBotService), // botServices будет заполнен после создания UseCase
		telegramClients,
		requestRepo, // для получения request по ID в HandleRAGResponse
		a.Log,
	)

	// Инициализируем астро-API клиент и сервис
	if a.Cfg.AstroAPI == nil {
		return nil, fmt.Errorf("astro API configuration is required: set TG_BOTS_ASTRO_API_* environment variables")
	}

	astroAPIClient := astroApiAdapter.NewClient(
		a.Cfg.AstroAPI,
		a.Log,
	)
	astroAPIService := astroApiService.New(astroAPIClient)

	// Инициализируем Kafka producers
	kafkaProducers := make(map[string]*kafkaAdapter.Producer)
	var ragProducer *kafkaAdapter.Producer
	for _, kafkaCfg := range a.Cfg.Kafka.List {
		if kafkaCfg.Config.Topic != "" && kafkaCfg.Config.ConsumerGroup == "" {
			// Это producer (есть topic, но нет consumer group)
			prod, err := kafkaAdapter.NewProducer(kafkaCfg.Config, a.Log)
			if err != nil {
				a.Log.Warn("failed to create kafka producer",
					"error", err,
					"name", kafkaCfg.Name,
				)
				continue
			}
			kafkaProducers[kafkaCfg.Name] = prod
			if kafkaCfg.Name == "rag_requests" {
				ragProducer = prod
			}
		}
	}

	// Создаём UseCase с Telegram Service, Astro API Service и Kafka Producer
	astroUseCase := astroUsecase.New(
		userRepo,
		requestRepo,
		statusRepo,
		tgService,
		astroAPIService,
		ragProducer, // может быть nil, если не настроен
		a.Log,
	)

	// Собираем все UseCase в map и обновляем Telegram Service
	botServicesMap := map[domain.BotType]service.IBotService{
		domain.BotTypeAstro: astroUseCase,
	}
	tgService.SetBotServices(botServicesMap)

	testService := testService.New(testRepo, a.Log)

	healthCheck := healthcheckController.New(db, a.Log)
	testController := testController.New(testService, a.Log)
	telegramController := telegramController.New(tgService, a.Log)

	httpServer := server.NewHTTPServer(
		a.Cfg.Server,
		a.Log,
		healthCheck,
		testController,
		telegramController,
	)

	// Инициализируем webhook или polling
	a.Log.Info("telegram configuration",
		"use_webhook", a.Cfg.Telegram.IsWebhookEnabled(),
		"webhook_url", a.Cfg.Telegram.WebhookURL,
	)

	var poller *tgAdapter.Poller
	if a.Cfg.Telegram.IsWebhookEnabled() {
		// Устанавливаем webhook для каждого бота при старте
		if err := a.setupWebhooks(ctx, telegramClients); err != nil {
			return nil, fmt.Errorf("failed to setup webhooks: %w", err)
		}
	} else {
		// Polling для локальной разработки
		a.Log.Warn("polling mode enabled - this should only be used for local development")
		poller = a.initPolling(tgService, telegramClients)
	}

	// Инициализируем Kafka consumers
	kafkaConsumers := make(map[string]*kafkaConsumerAdapter.Consumer)
	for _, kafkaCfg := range a.Cfg.Kafka.List {
		if kafkaCfg.Config.ConsumerGroup != "" {
			// Это consumer (есть consumer group)
			handler := a.createHandlerForTopic(kafkaCfg.Name, tgService)
			if handler == nil {
				a.Log.Warn("no handler for kafka topic, skipping consumer",
					"name", kafkaCfg.Name,
				)
				continue
			}
			consumer, err := kafkaConsumerAdapter.NewConsumer(kafkaCfg.Config, handler, a.Log)
			if err != nil {
				a.Log.Warn("failed to create kafka consumer",
					"error", err,
					"name", kafkaCfg.Name,
				)
				continue
			}
			kafkaConsumers[kafkaCfg.Name] = consumer
		}
	}

	return &Dependencies{
		DB:              db,
		HTTPServer:      httpServer,
		TelegramService: tgService,
		TelegramClients: telegramClients,
		TelegramPoller:  poller,
		KafkaProducers:  kafkaProducers,
		KafkaConsumers:  kafkaConsumers,
	}, nil
}

// setupWebhooks устанавливает webhook для всех ботов
func (a *App) setupWebhooks(ctx context.Context, telegramClients map[domain.BotId]*tgAdapter.Client) error {
	if a.Cfg.Telegram.WebhookURL == "" {
		return fmt.Errorf("webhook_url is required when use_webhook is true")
	}

	webhookURL := fmt.Sprintf("%s/webhook", a.Cfg.Telegram.WebhookURL)

	for botID, client := range telegramClients {
		// secret_token = bot_id (наш внутренний идентификатор)
		if err := client.SetWebhook(ctx, webhookURL, string(botID)); err != nil {
			a.Log.Error("failed to set webhook",
				"error", err,
				"bot_id", botID,
				"webhook_url", webhookURL,
			)
			return fmt.Errorf("failed to set webhook for bot %s: %w", botID, err)
		}

		a.Log.Info("webhook set successfully",
			"bot_id", botID,
			"webhook_url", webhookURL,
		)
	}

	return nil
}

// initPolling инициализирует polling для локальной разработки
func (a *App) initPolling(
	tgService *telegramService.Service,
	telegramClients map[domain.BotId]*tgAdapter.Client,
) *tgAdapter.Poller {
	handler := func(ctx context.Context, botID domain.BotId, update *domain.Update) error {
		return tgService.HandleUpdate(ctx, botID, update)
	}

	firstBotCfg := a.Cfg.Bots.List[0]
	firstBotID, _, _ := firstBotCfg.ToDomain()

	return tgAdapter.NewPoller(
		telegramClients[firstBotID],
		firstBotID,
		a.Cfg.Telegram,
		handler,
		a.Log,
	)
}

// registerBotCommands регистрирует команды бота в Telegram
func (a *App) registerBotCommands(ctx context.Context, client *tgAdapter.Client) error {
	commands := []tgAdapter.BotCommand{
		{Command: "start", Description: "Начать работу с ботом"},
		{Command: "help", Description: "Показать справку"},
		{Command: "my_info", Description: "Моя информация"},
		{Command: "reset_birth_data", Description: "Сбросить дату рождения"},
	}

	return client.SetMyCommands(ctx, commands)
}

func (a *App) initPostgres() (*sqlx.DB, error) {
	db, err := a.Cfg.Postgres.NewConnection()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	a.Log.Info("postgres connected successfully")

	if err := pg.RunMigrations(db, a.Log); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

// createHandlerForTopic создаёт handler для указанного топика
func (a *App) createHandlerForTopic(
	topicName string,
	tgService *telegramService.Service,
) kafka.MessageHandler {
	switch topicName {
	case "rag_responses":
		return kafkaHandlers.NewRAGResponseHandler(tgService, a.Log)
	default:
		a.Log.Warn("unknown kafka topic, using default handler",
			"topic", topicName,
		)
		return nil
	}
}
