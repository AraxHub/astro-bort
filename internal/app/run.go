package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"
)

func (a *App) runServices(ctx context.Context, deps *Dependencies) error {
	g, gCtx := errgroup.WithContext(ctx)

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

	// Telegram Updates: либо Webhook (prod), либо Polling (local dev)
	if a.Cfg.Telegram.IsWebhookEnabled() {
		a.Log.Info("telegram updates mode: webhook (production)",
			"webhook_url", a.Cfg.Telegram.WebhookURL)
	} else {
		g.Go(func() error {
			return a.runPolling(gCtx, deps)
		})
	}

	// Запускаем все Kafka consumers
	for name, consumer := range deps.KafkaConsumers {
		name := name // для замыкания в goroutine
		consumer := consumer
		g.Go(func() error {
			a.Log.Info("starting kafka consumer", "name", name)
			return consumer.Start(gCtx)
		})
	}

	// Запускаем планировщик джоб (запускает горутины внутри, сам не блокирует)
	if deps.JobScheduler != nil {
		a.Log.Info("starting job scheduler")
		if err := deps.JobScheduler.Start(gCtx); err != nil {
			a.Log.Error("failed to start job scheduler", "error", err)
		}
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

		// Закрываем Redis кэш
		if deps.Cache != nil {
			if err := deps.Cache.Close(); err != nil {
				a.Log.Error("failed to close cache", "error", err)
			}
		}

		// Закрываем Kafka producers
		for name, producer := range deps.KafkaProducers {
			if err := producer.Close(); err != nil {
				a.Log.Error("failed to close kafka producer", "error", err, "name", name)
			}
		}

		// Закрываем Kafka consumers
		for name, consumer := range deps.KafkaConsumers {
			if err := consumer.Close(); err != nil {
				a.Log.Error("failed to close kafka consumer", "error", err, "name", name)
			}
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

// runPolling запускает polling для локальной разработки
func (a *App) runPolling(ctx context.Context, deps *Dependencies) error {
	if deps.TelegramPoller == nil {
		return fmt.Errorf("telegram poller is not initialized")
	}

	// Удаляем webhook перед запуском polling
	deleteCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := deps.TelegramPoller.DeleteWebhook(deleteCtx); err != nil {
		a.Log.Warn("failed to delete webhook, continuing anyway", "error", err)
		// Ждём немного перед запуском polling, чтобы дать время на удаление webhook
		time.Sleep(2 * time.Second)
	} else {
		a.Log.Info("webhook deleted successfully, starting polling")
	}

	// Запускаем polling (botID уже сохранён в Poller)
	return deps.TelegramPoller.Start(ctx)
}
