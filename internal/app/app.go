package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"log/slog"

	server "github.com/admin/tg-bots/astro-bot/internal/adapters/primary/http"
	healthcheckController "github.com/admin/tg-bots/astro-bot/internal/adapters/primary/http/controllers/healthcheck"
	telegramController "github.com/admin/tg-bots/astro-bot/internal/adapters/primary/http/controllers/telegram"
	testController "github.com/admin/tg-bots/astro-bot/internal/adapters/primary/http/controllers/test"
	"github.com/admin/tg-bots/astro-bot/internal/adapters/secondary/storage/pg"
	tgAdapter "github.com/admin/tg-bots/astro-bot/internal/adapters/secondary/telegram"
	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/admin/tg-bots/astro-bot/internal/pkg/logger"
	"github.com/admin/tg-bots/astro-bot/internal/ports/service"
	requestRepo "github.com/admin/tg-bots/astro-bot/internal/repository/request"
	statusRepo "github.com/admin/tg-bots/astro-bot/internal/repository/status"
	testRepo "github.com/admin/tg-bots/astro-bot/internal/repository/test"
	userRepo "github.com/admin/tg-bots/astro-bot/internal/repository/user"
	telegramService "github.com/admin/tg-bots/astro-bot/internal/services/telegram"
	astroUsecase "github.com/admin/tg-bots/astro-bot/internal/usecases/astro"
	testService "github.com/admin/tg-bots/astro-bot/internal/usecases/test"
	"golang.org/x/sync/errgroup"

	"github.com/jmoiron/sqlx"
)

type App struct {
	Name string
	Cfg  *Config
	Log  *slog.Logger
}

func New(name string, cfg *Config) *App {
	return &App{
		Name: name,
		Cfg:  cfg,
		Log:  logger.New(name, cfg.Log),
	}
}

type Dependencies struct {
	DB              *sqlx.DB
	HTTPServer      *http.Server
	TelegramService *telegramService.Service
	TelegramClient  *tgAdapter.Client
	TelegramPoller  *tgAdapter.Poller
}

func (a *App) Run(ctx context.Context) error {
	a.Log.Info("starting application")

	// Инициализация зависимостей
	deps, err := a.initDependencies(ctx)
	if err != nil {
		return fmt.Errorf("failed to init dependencies: %w", err)
	}

	// Запуск сервисов
	return a.runServices(ctx, deps)
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

	telegramClient := tgAdapter.NewClient(a.Cfg.Telegram.BotToken, a.Log)

	// Регистрируем команды бота
	if err := a.registerBotCommands(ctx, telegramClient); err != nil {
		a.Log.Warn("failed to register bot commands", "error", err)
		// Не критично, продолжаем работу
	}

	astroUseCase := astroUsecase.New(
		userRepo,
		requestRepo,
		statusRepo,
		telegramClient,
		a.Log,
	)

	botServicesMap := map[string]service.IBotService{
		"astro": astroUseCase,
	}
	telegramService := telegramService.New(
		botServicesMap,
		telegramClient,
		a.Log,
	)

	testService := testService.New(testRepo, a.Log)

	healthCheck := healthcheckController.New(db, a.Log)
	testController := testController.New(testService, a.Log)
	telegramController := telegramController.New(telegramService, a.Log)

	httpServer := server.NewHTTPServer(
		a.Cfg.Server,
		a.Log,
		healthCheck,
		testController,
		telegramController,
	)

	var poller *tgAdapter.Poller
	if !a.Cfg.Telegram.UseWebhook {
		handler := func(ctx context.Context, botID string, update *domain.Update) error {
			return telegramService.HandleUpdate(ctx, botID, update)
		}
		poller = tgAdapter.NewPoller(
			telegramClient,
			a.Cfg.Telegram,
			handler,
			a.Log,
		)
	}

	return &Dependencies{
		DB:              db,
		HTTPServer:      httpServer,
		TelegramService: telegramService,
		TelegramClient:  telegramClient,
		TelegramPoller:  poller,
	}, nil
}

func (a *App) runServices(ctx context.Context, deps *Dependencies) error {
	g, gCtx := errgroup.WithContext(ctx)

	// HTTP Server
	g.Go(func() error {
		a.Log.Info("starting http server",
			"host", a.Cfg.Server.Host,
			"port", a.Cfg.Server.Port)

		err := deps.HTTPServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("http server error: %w", err)
		}
		return nil
	})

	// Telegram Polling (если webhook выключен)
	if deps.TelegramPoller != nil {
		g.Go(func() error {
			a.Log.Info("starting telegram polling",
				"bot_token", maskToken(a.Cfg.Telegram.BotToken))

			// Удаляем webhook перед запуском polling
			// Используем отдельный контекст с таймаутом для удаления webhook
			deleteCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if err := deps.TelegramPoller.DeleteWebhook(deleteCtx); err != nil {
				a.Log.Warn("failed to delete webhook, continuing anyway", "error", err)
				// Ждём немного перед запуском polling, чтобы дать время на удаление webhook
				time.Sleep(2 * time.Second)
			} else {
				a.Log.Info("webhook deleted successfully, starting polling")
			}

			// Запускаем polling
			botID := "astro" // TODO: получать из конфига
			return deps.TelegramPoller.Start(gCtx, botID)
		})
	}

	// Graceful shutdown
	g.Go(func() error {
		<-gCtx.Done()
		a.Log.Info("received shutdown signal")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := deps.HTTPServer.Shutdown(shutdownCtx); err != nil {
			a.Log.Error("failed to shutdown http server", "error", err)
		}

		if err := deps.DB.Close(); err != nil {
			a.Log.Error("failed to close database", "error", err)
		}

		a.Log.Info("application shutdown completed")
		return nil
	})

	if err := g.Wait(); err != nil {
		a.Log.Error("application error", "error", err)
		return err
	}

	return nil
}

// maskToken маскирует токен для логирования
func maskToken(token string) string {
	if len(token) <= 8 {
		return "***"
	}
	return token[:4] + "..." + token[len(token)-4:]
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
