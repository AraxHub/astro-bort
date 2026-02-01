package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"
)

// runServices запускает все сервисы приложения и управляет их жизненным циклом
func (a *App) runServices(ctx context.Context, deps *Dependencies) error {
	g, gCtx := errgroup.WithContext(ctx)

	// HTTP сервер
	g.Go(a.runHTTPServer(deps))

	// Telegram: webhook или polling
	a.runTelegramUpdates(gCtx, g, deps)

	// Kafka consumers
	a.runKafkaConsumers(gCtx, g, deps)

	// Job scheduler
	a.runJobScheduler(gCtx, g, deps)

	// Graceful shutdown
	g.Go(a.runGracefulShutdown(gCtx, deps))

	if err := g.Wait(); err != nil {
		a.Log.Error("application error", "error", err)
		return err
	}

	return nil
}

// runHTTPServer запускает HTTP сервер
func (a *App) runHTTPServer(deps *Dependencies) func() error {
	return func() error {
		a.Log.Info("starting http server", "host", a.Cfg.Server.Host, "port", a.Cfg.Server.Port)

		err := deps.HTTPServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("http server error: %w", err)
		}

		return nil
	}
}

// runTelegramUpdates запускает Telegram updates (webhook или polling)
func (a *App) runTelegramUpdates(ctx context.Context, g *errgroup.Group, deps *Dependencies) {
	if a.Cfg.Telegram.IsWebhookEnabled() {
		a.Log.Info("telegram updates mode: webhook (production)", "webhook_url", a.Cfg.Telegram.WebhookURL)
		return
	}

	g.Go(func() error {
		return a.runPolling(ctx, deps)
	})
}

// runKafkaConsumers запускает все Kafka consumers
func (a *App) runKafkaConsumers(ctx context.Context, g *errgroup.Group, deps *Dependencies) {
	for name, consumer := range deps.KafkaConsumers {
		name := name // для замыкания в goroutine
		consumer := consumer

		g.Go(func() error {
			a.Log.Info("starting kafka consumer", "name", name)
			return consumer.Start(ctx)
		})
	}
}

// runJobScheduler запускает планировщик джоб
func (a *App) runJobScheduler(ctx context.Context, g *errgroup.Group, deps *Dependencies) {
	if deps.JobScheduler == nil {
		return
	}

	g.Go(func() error {
		a.Log.Info("starting job scheduler")
		if err := deps.JobScheduler.Start(ctx); err != nil {
			a.Log.Error("failed to start job scheduler", "error", err)
			return fmt.Errorf("job scheduler error: %w", err)
		}
		// Планировщик работает в фоне, но мы отслеживаем его запуск
		<-ctx.Done()
		a.Log.Info("job scheduler stopped")
		return nil
	})
}

// runGracefulShutdown обрабатывает graceful shutdown всех сервисов
func (a *App) runGracefulShutdown(ctx context.Context, deps *Dependencies) func() error {
	return func() error {
		<-ctx.Done()
		a.Log.Info("received shutdown signal")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		a.shutdownServices(shutdownCtx, deps)
		a.Log.Info("application shutdown completed")

		return nil
	}
}

// shutdownServices закрывает все сервисы и соединения
func (a *App) shutdownServices(ctx context.Context, deps *Dependencies) {
	if err := deps.HTTPServer.Shutdown(ctx); err != nil {
		a.Log.Error("failed to shutdown http server", "error", err)
	}

	if err := deps.DB.Close(); err != nil {
		a.Log.Error("failed to close database", "error", err)
	}

	if deps.Cache != nil {
		if err := deps.Cache.Close(); err != nil {
			a.Log.Error("failed to close cache", "error", err)
		}
	}

	for name, producer := range deps.KafkaProducers {
		if err := producer.Close(); err != nil {
			a.Log.Error("failed to close kafka producer", "error", err, "name", name)
		}
	}

	for name, consumer := range deps.KafkaConsumers {
		if err := consumer.Close(); err != nil {
			a.Log.Error("failed to close kafka consumer", "error", err, "name", name)
		}
	}
}

// runPolling запускает polling для локальной разработки
func (a *App) runPolling(ctx context.Context, deps *Dependencies) error {
	if deps.TelegramPoller == nil {
		return fmt.Errorf("telegram poller is not initialized")
	}

	deleteCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := deps.TelegramPoller.DeleteWebhook(deleteCtx); err != nil {
		a.Log.Warn("failed to delete webhook, continuing anyway", "error", err)
		time.Sleep(2 * time.Second) // даём время на удаление webhook
	} else {
		a.Log.Info("webhook deleted successfully, starting polling")
	}

	return deps.TelegramPoller.Start(ctx)
}
