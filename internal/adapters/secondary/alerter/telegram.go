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
	telegramClient  *telegram.Client
	chatID          int64
	messageThreadID *int64
	log             *slog.Logger
}

// NewClient создаёт новый клиент для отправки алертов
func NewClient(cfg *Config, log *slog.Logger) *Client {
	if cfg == nil {
		return nil
	}

	tgClient := telegram.NewClient(cfg.BotToken, log)
	return &Client{
		telegramClient:  tgClient,
		chatID:          cfg.ChatID,
		messageThreadID: cfg.MessageThreadID,
		log:             log,
	}
}

// SendAlert отправляет алерт в Telegram группу (или топик форума)
func (c *Client) SendAlert(ctx context.Context, message string) error {
	if c == nil || c.telegramClient == nil {
		return fmt.Errorf("alerter client is not initialized")
	}

	if err := c.sendMessageToTopic(ctx, c.chatID, message, c.messageThreadID); err != nil {
		c.log.Warn("failed to send alert",
			"error", err,
			"chat_id", c.chatID,
			"message_thread_id", c.messageThreadID,
		)
		return fmt.Errorf("failed to send alert: %w", err)
	}

	c.log.Debug("alert sent successfully",
		"chat_id", c.chatID,
		"message_thread_id", c.messageThreadID,
	)

	return nil
}

// sendMessageToTopic отправляет сообщение в чат или топик форума
func (c *Client) sendMessageToTopic(ctx context.Context, chatID int64, text string, threadID *int64) error {
	req := telegram.SendMessageRequest{
		ChatID:          chatID,
		Text:            text,
		MessageThreadID: threadID,
	}

	_, err := c.telegramClient.SendMessageWithRequest(ctx, req)
	return err
}
