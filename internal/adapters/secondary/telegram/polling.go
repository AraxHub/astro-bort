package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"log/slog"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
)

// UpdateHandler функция для обработки обновлений от Telegram
type UpdateHandler func(ctx context.Context, botID string, update *domain.Update) error

// Poller реализует long polling для получения обновлений от Telegram
type Poller struct {
	client       *Client
	config       *Config
	handler      UpdateHandler
	lastUpdateID int64
	log          *slog.Logger
	httpClient   *http.Client // отдельный HTTP клиент с увеличенным таймаутом для polling
}

func NewPoller(client *Client, config *Config, handler UpdateHandler, log *slog.Logger) *Poller {
	pollingTimeout := config.PollingTimeout
	if pollingTimeout <= 0 {
		pollingTimeout = 30
	}
	// HTTP таймаут = polling timeout + запас (10 секунд)
	httpTimeout := time.Duration(pollingTimeout+10) * time.Second

	return &Poller{
		client:       client,
		config:       config,
		handler:      handler,
		lastUpdateID: 0,
		log:          log,
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
	}
}

// UpdateResult структура для одного обновления из getUpdates
type UpdateResult struct {
	UpdateID      int64                 `json:"update_id"`
	Message       *domain.Message       `json:"message,omitempty"`
	CallbackQuery *domain.CallbackQuery `json:"callback_query,omitempty"`
}

// GetUpdatesResponse ответ от Telegram API для getUpdates
type GetUpdatesResponse struct {
	APIResponse
	Result []UpdateResult `json:"result"`
}

// Start запускает long polling в отдельной горутине
func (p *Poller) Start(ctx context.Context, botID string) error {
	p.log.Info("starting telegram polling",
		"bot_id", botID,
		"timeout", p.config.PollingTimeout,
	)

	for {
		select {
		case <-ctx.Done():
			p.log.Info("polling stopped", "bot_id", botID)
			return ctx.Err()
		default:
			updates, err := p.getUpdates(ctx)
			if err != nil {
				p.log.Error("failed to get updates",
					"error", err,
					"bot_id", botID,
				)
				// Ждём перед повтором
				time.Sleep(5 * time.Second)
				continue
			}

			for _, updateData := range updates {
				update := &domain.Update{
					UpdateID:      updateData.UpdateID,
					Message:       updateData.Message,
					CallbackQuery: updateData.CallbackQuery,
				}

				// Обновляем lastUpdateID
				if updateData.UpdateID >= p.lastUpdateID {
					p.lastUpdateID = updateData.UpdateID + 1
				}

				// Обрабатываем обновление через handler
				if err := p.handler(ctx, botID, update); err != nil {
					p.log.Error("failed to handle update",
						"error", err,
						"update_id", update.UpdateID,
						"bot_id", botID,
					)
					// Продолжаем обработку следующих обновлений
				}
			}
		}
	}
}

// getUpdates получает обновления от Telegram API
func (p *Poller) getUpdates(ctx context.Context) ([]UpdateResult, error) {
	timeout := p.config.PollingTimeout
	if timeout <= 0 {
		timeout = 30 // дефолтный таймаут
	}

	url := fmt.Sprintf("%s/getUpdates?offset=%d&timeout=%d", p.client.baseURL, p.lastUpdateID, timeout)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResp GetUpdatesResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		p.log.Error("failed to unmarshal response",
			"error", err,
			"status_code", resp.StatusCode,
			"body", string(body),
		)
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !apiResp.OK {
		// Ошибка 409 - конфликт (другой экземпляр бота или webhook активен)
		if apiResp.ErrorCode == 409 {
			p.log.Warn("telegram API conflict - another bot instance or webhook is active",
				"error_code", apiResp.ErrorCode,
				"description", apiResp.Description,
			)
			// Возвращаем пустой список обновлений, чтобы не прерывать цикл
			// В следующей итерации попробуем снова
			return []UpdateResult{}, nil
		}

		p.log.Error("telegram API returned error",
			"error_code", apiResp.ErrorCode,
			"description", apiResp.Description,
			"status_code", resp.StatusCode,
		)
		return nil, fmt.Errorf("telegram API error: %s (code: %d)", apiResp.Description, apiResp.ErrorCode)
	}

	return apiResp.Result, nil
}

// DeleteWebhook удаляет webhook (нужно вызывать отдельно перед запуском polling)
func (p *Poller) DeleteWebhook(ctx context.Context) error {
	url := p.client.baseURL + "/deleteWebhook?drop_pending_updates=true"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		p.log.Warn("deleteWebhook returned non-OK status",
			"status_code", resp.StatusCode,
			"body", string(body),
		)
		return fmt.Errorf("deleteWebhook failed with status %d", resp.StatusCode)
	}

	p.log.Info("webhook deleted successfully")
	return nil
}
