package app

import (
	"context"
	"fmt"
	"net/http"

	server "github.com/admin/tg-bots/astro-bot/internal/adapters/primary/http"
	alerterController "github.com/admin/tg-bots/astro-bot/internal/adapters/primary/http/controllers/alerter"
	healthcheckController "github.com/admin/tg-bots/astro-bot/internal/adapters/primary/http/controllers/healthcheck"
	telegramController "github.com/admin/tg-bots/astro-bot/internal/adapters/primary/http/controllers/telegram"
	kafkaConsumerAdapter "github.com/admin/tg-bots/astro-bot/internal/adapters/primary/kafka"
	kafkaHandlers "github.com/admin/tg-bots/astro-bot/internal/adapters/primary/kafka/handlers"
	alerterAdapter "github.com/admin/tg-bots/astro-bot/internal/adapters/secondary/alerter"
	astroApiAdapter "github.com/admin/tg-bots/astro-bot/internal/adapters/secondary/astroApi"
	kafkaAdapter "github.com/admin/tg-bots/astro-bot/internal/adapters/secondary/kafka"
	"github.com/admin/tg-bots/astro-bot/internal/adapters/secondary/storage/pg"
	redisAdapter "github.com/admin/tg-bots/astro-bot/internal/adapters/secondary/storage/redis"
	tgAdapter "github.com/admin/tg-bots/astro-bot/internal/adapters/secondary/telegram"
	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/admin/tg-bots/astro-bot/internal/ports/cache"
	"github.com/admin/tg-bots/astro-bot/internal/ports/kafka"
	"github.com/admin/tg-bots/astro-bot/internal/ports/repository"
	"github.com/admin/tg-bots/astro-bot/internal/ports/service"
	paymentRepo "github.com/admin/tg-bots/astro-bot/internal/repository/payment"
	requestRepo "github.com/admin/tg-bots/astro-bot/internal/repository/request"
	statusRepo "github.com/admin/tg-bots/astro-bot/internal/repository/status"
	userRepo "github.com/admin/tg-bots/astro-bot/internal/repository/user"
	alerterService "github.com/admin/tg-bots/astro-bot/internal/services/alerter"
	astroApiService "github.com/admin/tg-bots/astro-bot/internal/services/astroApi"
	jobScheduler "github.com/admin/tg-bots/astro-bot/internal/services/jobs"
	telegramService "github.com/admin/tg-bots/astro-bot/internal/services/telegram"
	astroUsecase "github.com/admin/tg-bots/astro-bot/internal/usecases/astro"
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
	Cache           cache.Cache
	JobScheduler    *jobScheduler.Scheduler
}

// initDependencies инициализирует все зависимости приложения
func (a *App) initDependencies(ctx context.Context) (*Dependencies, error) {
	db, err := a.initPostgres()
	if err != nil {
		return nil, fmt.Errorf("failed to init postgres: %w", err)
	}

	repos := a.initRepositories(db)
	telegramClients, tgService, err := a.initTelegram(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to init telegram: %w", err)
	}

	externalServices := a.initExternalServices()
	kafkaProducers, kafkaConsumers, err := a.initKafka(ctx, tgService)
	if err != nil {
		return nil, fmt.Errorf("failed to init kafka: %w", err)
	}

	astroUseCase := a.initUseCases(repos, tgService, externalServices, kafkaProducers)
	tgService.SetBotServices(map[domain.BotType]service.IBotService{
		domain.BotTypeAstro: astroUseCase,
	})

	// Инициализируем payment use case и интегрируем в telegram service и astro use case
	a.initPayment(telegramClients, repos, tgService, externalServices.Alerter, astroUseCase)

	httpServer := a.initHTTP(db, tgService, externalServices.Alerter)
	poller, err := a.initTelegramMode(ctx, tgService, telegramClients)
	if err != nil {
		return nil, fmt.Errorf("failed to init telegram mode: %w", err)
	}

	scheduler := a.initJobScheduler(externalServices.Alerter, astroUseCase, externalServices.Cache, repos)

	return &Dependencies{
		DB:              db,
		HTTPServer:      httpServer,
		TelegramService: tgService,
		TelegramClients: telegramClients,
		TelegramPoller:  poller,
		KafkaProducers:  kafkaProducers,
		KafkaConsumers:  kafkaConsumers,
		Cache:           externalServices.Cache,
		JobScheduler:    scheduler,
	}, nil
}

// repositories содержит инициализированные репозитории
type repositories struct {
	User    repository.IUserRepo
	Request repository.IRequestRepo
	Status  repository.IStatusRepo
	Payment repository.IPaymentRepo
}

// initRepositories инициализирует репозитории для работы с БД
func (a *App) initRepositories(db *sqlx.DB) *repositories {
	persistenceLayer := pg.NewDB(db)
	return &repositories{
		User:    userRepo.New(persistenceLayer, a.Log),
		Request: requestRepo.New(persistenceLayer, a.Log),
		Status:  statusRepo.New(persistenceLayer, a.Log),
		Payment: paymentRepo.New(persistenceLayer, a.Log),
	}
}

// externalServices содержит внешние сервисы (опциональные)
type externalServices struct {
	AstroAPI service.IAstroAPIService
	Alerter  service.IAlerterService
	Cache    cache.Cache
}

// initExternalServices инициализирует внешние сервисы (AstroAPI, Alerter, Cache)
func (a *App) initExternalServices() *externalServices {
	services := &externalServices{}

	// AstroAPI - обязательный
	if a.Cfg.AstroAPI == nil {
		a.Log.Warn("astro API configuration is missing")
	} else {
		astroAPIClient := astroApiAdapter.NewClient(a.Cfg.AstroAPI, a.Log)
		services.AstroAPI = astroApiService.New(astroAPIClient)
	}

	// Alerter - опциональный
	if a.Cfg.Alerter != nil {
		alerterClient := alerterAdapter.NewClient(a.Cfg.Alerter, a.Log)
		services.Alerter = alerterService.New(alerterClient)
	}

	// Redis Cache - опциональный
	if a.Cfg.Redis != nil {
		redisClient, err := a.Cfg.Redis.NewConnection()
		if err != nil {
			a.Log.Warn("failed to init redis cache, continuing without cache", "error", err)
		} else {
			services.Cache = redisAdapter.NewClient(redisClient)
			a.Log.Info("redis cache connected successfully")
		}
	}

	return services
}

// initTelegram инициализирует Telegram клиенты и сервис
func (a *App) initTelegram(ctx context.Context) (
	clients map[domain.BotId]*tgAdapter.Client,
	tgSvc *telegramService.Service,
	err error,
) {
	if len(a.Cfg.Bots.List) == 0 {
		return nil, nil, fmt.Errorf("no bots configured: at least one bot must be specified via BOTS_COUNT and BOTS_0_* environment variables")
	}

	botIDToType := make(map[domain.BotId]domain.BotType)
	clients = make(map[domain.BotId]*tgAdapter.Client)

	for i, botCfg := range a.Cfg.Bots.List {
		botID, botType, err := botCfg.ToDomain()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to convert bot config at index %d: %w", i, err)
		}

		botIDToType[botID] = botType
		clients[botID] = tgAdapter.NewClient(botCfg.BotToken, a.Log)

		if err := a.registerBotCommands(ctx, clients[botID]); err != nil {
			a.Log.Warn("failed to register bot commands", "error", err, "bot_id", botID)
		}
	}

	tgSvc = telegramService.New(
		botIDToType,
		make(map[domain.BotType]service.IBotService), // будет заполнен после создания UseCase
		clients,
		a.Log,
	)

	return clients, tgSvc, nil
}

// initKafka инициализирует Kafka producers и consumers
func (a *App) initKafka(
	ctx context.Context,
	tgService *telegramService.Service,
) (
	producers map[string]*kafkaAdapter.Producer,
	consumers map[string]*kafkaConsumerAdapter.Consumer,
	err error,
) {
	producers = make(map[string]*kafkaAdapter.Producer)
	consumers = make(map[string]*kafkaConsumerAdapter.Consumer)

	for _, kafkaCfg := range a.Cfg.Kafka.List {
		// Producer: есть topic, но нет consumer group
		if kafkaCfg.Config.Topic != "" && kafkaCfg.Config.ConsumerGroup == "" {
			prod, err := kafkaAdapter.NewProducer(kafkaCfg.Config, a.Log)
			if err != nil {
				a.Log.Warn("failed to create kafka producer", "error", err, "name", kafkaCfg.Name)
				continue
			}
			producers[kafkaCfg.Name] = prod
		}

		// Consumer: есть consumer group
		if kafkaCfg.Config.ConsumerGroup != "" {
			handler := a.createHandlerForTopic(kafkaCfg.Name, tgService)
			if handler == nil {
				a.Log.Warn("no handler for kafka topic, skipping consumer", "name", kafkaCfg.Name)
				continue
			}

			consumer, err := kafkaConsumerAdapter.NewConsumer(kafkaCfg.Config, handler, a.Log)
			if err != nil {
				a.Log.Warn("failed to create kafka consumer", "error", err, "name", kafkaCfg.Name)
				continue
			}
			consumers[kafkaCfg.Name] = consumer
		}
	}

	return producers, consumers, nil
}

// initUseCases инициализирует UseCases приложения
func (a *App) initUseCases(
	repos *repositories,
	tgService *telegramService.Service,
	externalServices *externalServices,
	kafkaProducers map[string]*kafkaAdapter.Producer,
) *astroUsecase.Service {
	var ragProducer *kafkaAdapter.Producer
	if prod, ok := kafkaProducers["requests"]; ok {
		ragProducer = prod
	}

	// дефолт
	freeMessagesLimit := 15
	starsPrice := int64(1000)
	if a.Cfg.Astro != nil {
		freeMessagesLimit = a.Cfg.Astro.FreeMessagesLimit
		if a.Cfg.Astro.StarsPrice > 0 {
			starsPrice = a.Cfg.Astro.StarsPrice
		}
	}

	return astroUsecase.New(
		repos.User,
		repos.Request,
		repos.Status,
		tgService,
		externalServices.AstroAPI,
		ragProducer,              // может быть nil
		externalServices.Alerter, // может быть nil
		externalServices.Cache,   // может быть nil
		freeMessagesLimit,
		starsPrice,
		a.Log,
	)
}

// initHTTP инициализирует HTTP сервер и контроллеры
func (a *App) initHTTP(
	db *sqlx.DB,
	tgService *telegramService.Service,
	alerterSvc service.IAlerterService,
) *http.Server {
	controllers := []server.Controller{
		healthcheckController.New(db, a.Log),
		telegramController.New(tgService, a.Log),
	}

	if alerterSvc != nil {
		controllers = append(controllers, alerterController.New(alerterSvc, a.Log))
	}

	return server.NewHTTPServer(a.Cfg.Server, a.Log, controllers...)
}

// initTelegramMode инициализирует режим работы Telegram (webhook или polling)
func (a *App) initTelegramMode(
	ctx context.Context,
	tgService *telegramService.Service,
	telegramClients map[domain.BotId]*tgAdapter.Client,
) (*tgAdapter.Poller, error) {
	a.Log.Info("telegram configuration",
		"use_webhook", a.Cfg.Telegram.IsWebhookEnabled(),
		"webhook_url", a.Cfg.Telegram.WebhookURL,
	)

	if a.Cfg.Telegram.IsWebhookEnabled() {
		if err := a.setupWebhooks(ctx, telegramClients); err != nil {
			return nil, fmt.Errorf("failed to setup webhooks: %w", err)
		}
		return nil, nil // webhook режим, poller не нужен
	}

	a.Log.Warn("polling mode enabled - this should only be used for local development")
	return a.initPolling(tgService, telegramClients), nil
}

// initJobScheduler инициализирует планировщик джоб
func (a *App) initJobScheduler(
	alerterSvc service.IAlerterService,
	astroUseCase *astroUsecase.Service,
	cacheClient cache.Cache,
	repos *repositories,
) *jobScheduler.Scheduler {
	scheduler := jobScheduler.NewScheduler(a.Log, alerterSvc)

	// Регистрируем джобу для обновления позиций планет (если кеш включен)
	if cacheClient != nil {
		positionsUpdater := jobScheduler.NewPositionsUpdater(astroUseCase, a.Log)
		scheduler.Register(positionsUpdater)
		a.Log.Info("positions updater job registered")
	}

	// Регистрируем джобу для проверки истёкших подписок
	if astroUseCase != nil {
		subscriptionExpirer := jobScheduler.NewSubscriptionExpirer(astroUseCase, a.Log)
		scheduler.Register(subscriptionExpirer)
		a.Log.Info("subscription expirer job registered")

		// Регистрируем джобы для отправки пушей
		weeklyForecastPush := jobScheduler.NewWeeklyForecastPush(astroUseCase, a.Log)
		scheduler.Register(weeklyForecastPush)
		a.Log.Info("weekly forecast push job registered")

		situationalWarningPush := jobScheduler.NewSituationalWarningPush(astroUseCase, a.Log)
		scheduler.Register(situationalWarningPush)
		a.Log.Info("situational warning push job registered")

		premiumLimitPush := jobScheduler.NewPremiumLimitPush(astroUseCase, a.Log)
		scheduler.Register(premiumLimitPush)
		a.Log.Info("premium limit push job registered")
	}

	return scheduler
}

// setupWebhooks устанавливает webhook для всех ботов
func (a *App) setupWebhooks(ctx context.Context, telegramClients map[domain.BotId]*tgAdapter.Client) error {
	if a.Cfg.Telegram.WebhookURL == "" {
		return fmt.Errorf("webhook_url is required when use_webhook is true")
	}

	webhookURL := fmt.Sprintf("%s/webhook", a.Cfg.Telegram.WebhookURL)

	for botID, client := range telegramClients {
		if err := client.SetWebhook(ctx, webhookURL, string(botID)); err != nil {
			a.Log.Error("failed to set webhook", "error", err, "bot_id", botID, "webhook_url", webhookURL)
			return fmt.Errorf("failed to set webhook for bot %s: %w", botID, err)
		}

		a.Log.Info("webhook set successfully", "bot_id", botID, "webhook_url", webhookURL)
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

// initPostgres инициализирует подключение к PostgreSQL и запускает миграции
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

// createHandlerForTopic создаёт handler для указанного топика Kafka
func (a *App) createHandlerForTopic(
	topicName string,
	tgService *telegramService.Service,
) kafka.MessageHandler {
	switch topicName {
	case "responses":
		return kafkaHandlers.NewRAGResponseHandler(tgService, a.Log)
	default:
		a.Log.Warn("unknown kafka topic, using default handler", "topic", topicName)
		return nil
	}
}
