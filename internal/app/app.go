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
	testController "github.com/admin/tg-bots/astro-bot/internal/adapters/primary/http/controllers/test"
	"github.com/admin/tg-bots/astro-bot/internal/adapters/secondary/storage/pg"
	"github.com/admin/tg-bots/astro-bot/internal/pkg/logger"
	testRepo "github.com/admin/tg-bots/astro-bot/internal/repository/test"
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

func (a *App) Run(ctx context.Context) error {
	a.Log.Info("running astro-test")

	db, err := a.initPostgres()
	if err != nil {
		return fmt.Errorf("failed to init postgres: %w", err)
	}

	persistenceLayer := pg.NewDB(db)
	testRepo := testRepo.New(persistenceLayer, a.Log)
	testService := testService.New(testRepo, a.Log)
	testController := testController.New(testService, a.Log)
	healthCheck := healthcheckController.New(db, a.Log)

	httpServer := server.NewHTTPServer(a.Cfg.Server, a.Log, healthCheck, testController)

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		a.Log.Info("starting http server",
			"host", a.Cfg.Server.Host,
			"port", a.Cfg.Server.Port)

		err := httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("http server error: %w", err)
		}
		return nil
	})

	// Graceful shutdown
	g.Go(func() error {
		<-gCtx.Done()
		a.Log.Info("received shutdown signal")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			a.Log.Error("failed to shutdown http server", "error", err)
		}

		if err := db.Close(); err != nil {
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
