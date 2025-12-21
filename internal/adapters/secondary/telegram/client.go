package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"log/slog"
)

const (
	telegramAPIBaseURL = "https://api.telegram.org/bot"
	apiTimeout         = 30 * time.Second
)

// Client клиент для работы с Telegram Bot API
type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
	log        *slog.Logger
}

// NewClient создаёт новый клиент для Telegram Bot API
func NewClient(token string, log *slog.Logger) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: apiTimeout,
		},
		baseURL: telegramAPIBaseURL + token,
		token:   token,
		log:     log,
	}
}

// SendMessageRequest запрос на отправку сообщения
type SendMessageRequest struct {
	ChatID      int64                  `json:"chat_id"`
	Text        string                 `json:"text"`
	ParseMode   string                 `json:"parse_mode,omitempty"` // "HTML", "Markdown", "MarkdownV2"
	ReplyMarkup map[string]interface{} `json:"reply_markup,omitempty"`
}

// SendMessageResult результат отправки сообщения
type SendMessageResult struct {
	MessageID int64 `json:"message_id"`
	Chat      struct {
		ID int64 `json:"id"`
	} `json:"chat"`
	Text string `json:"text"`
	Date int64  `json:"date"`
}

// SendMessageResponse ответ от Telegram API
type SendMessageResponse struct {
	APIResponse
	Result SendMessageResult `json:"result"`
}

// SendMessage отправляет текстовое сообщение
func (c *Client) SendMessage(ctx context.Context, chatID int64, text string) error {
	req := SendMessageRequest{
		ChatID: chatID,
		Text:   text,
	}

	return c.sendMessage(ctx, req)
}

// SendMessageWithKeyboard отправляет сообщение с клавиатурой
func (c *Client) SendMessageWithKeyboard(ctx context.Context, chatID int64, text string, keyboard map[string]interface{}) error {
	req := SendMessageRequest{
		ChatID:      chatID,
		Text:        text,
		ReplyMarkup: keyboard,
	}

	return c.sendMessage(ctx, req)
}

// sendMessage выполняет запрос к Telegram API для отправки сообщения
func (c *Client) sendMessage(ctx context.Context, req SendMessageRequest) error {
	url := c.baseURL + "/sendMessage"

	jsonData, err := json.Marshal(req)
	if err != nil {
		c.log.Error("failed to marshal request",
			"error", err,
			"chat_id", req.ChatID,
		)
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		c.log.Error("failed to create request",
			"error", err,
			"chat_id", req.ChatID,
		)
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.log.Error("failed to send request to telegram",
			"error", err,
			"chat_id", req.ChatID,
		)
		return fmt.Errorf("failed to send request to telegram: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.log.Error("failed to read response body",
			"error", err,
			"chat_id", req.ChatID,
			"status_code", resp.StatusCode,
		)
		return fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResp SendMessageResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		c.log.Error("failed to unmarshal response",
			"error", err,
			"chat_id", req.ChatID,
			"status_code", resp.StatusCode,
			"body", string(body),
		)
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !apiResp.OK {
		c.log.Error("telegram API returned error",
			"error_code", apiResp.ErrorCode,
			"description", apiResp.Description,
			"chat_id", req.ChatID,
			"status_code", resp.StatusCode,
		)
		return fmt.Errorf("telegram API error: %s (code: %d)", apiResp.Description, apiResp.ErrorCode)
	}

	c.log.Debug("message sent successfully",
		"chat_id", req.ChatID,
		"message_id", apiResp.Result.MessageID,
	)

	return nil
}

// GetMe получает информацию о боте
func (c *Client) GetMe(ctx context.Context) error {
	url := c.baseURL + "/getMe"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.log.Error("getMe failed",
			"status_code", resp.StatusCode,
			"body", string(body),
		)
		return fmt.Errorf("getMe failed with status %d", resp.StatusCode)
	}

	c.log.Info("bot info retrieved successfully")
	return nil
}

// BotCommand представляет команду бота
type BotCommand struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

// SetMyCommands регистрирует команды бота в меню
func (c *Client) SetMyCommands(ctx context.Context, commands []BotCommand) error {
	reqBody := struct {
		Commands []BotCommand `json:"commands"`
	}{
		Commands: commands,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + "/setMyCommands"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		c.log.Error("failed to unmarshal response",
			"error", err,
			"status_code", resp.StatusCode,
			"body", string(body),
		)
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !apiResp.OK {
		c.log.Error("telegram API returned error",
			"error_code", apiResp.ErrorCode,
			"description", apiResp.Description,
			"status_code", resp.StatusCode,
		)
		return fmt.Errorf("telegram API error: %s (code: %d)", apiResp.Description, apiResp.ErrorCode)
	}

	c.log.Info("bot commands registered successfully", "commands_count", len(commands))
	return nil
}
