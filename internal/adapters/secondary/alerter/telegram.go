package alerter

import (
	"context"
	"fmt"

	"log/slog"

	"github.com/admin/tg-bots/astro-bot/internal/adapters/secondary/telegram"
)
//согл, что чистота нарушена, но тут выбор в пользу делегирования ответственности другому адаптеру

// Client клиент для отправки алертов через Telegram
type Client struct {
	telegramClient *telegram.Client
	chatID         int64
	log            *slog.Logger
}

// NewClient создаёт новый клиент для отправки алертов
func NewClient(cfg *Config, log *slog.Logger) *Client {
	if cfg == nil {
		return nil
	}

	tgClient := telegram.NewClient(cfg.BotToken, log)
	return &Client{
		telegramClient: tgClient,
		chatID:         cfg.ChatID,
		log:            log,
	}
}

// SendAlert отправляет алерт в Telegram группу
func (c *Client) SendAlert(ctx context.Context, message string) error {
	if c == nil || c.telegramClient == nil {
		return fmt.Errorf("alerter client is not initialized")
	}

	if err := c.telegramClient.SendMessage(ctx, c.chatID, message); err != nil {
		c.log.Warn("failed to send alert",
			"error", err,
			"chat_id", c.chatID,
		)
		return fmt.Errorf("failed to send alert: %w", err)
	}

	c.log.Debug("alert sent successfully",
		"chat_id", c.chatID,
	)

	return nil
}
