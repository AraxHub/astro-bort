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

// truncateString обрезает строку до указанной длины
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

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
	ChatID          int64                  `json:"chat_id"`
	Text            string                 `json:"text"`
	ParseMode       string                 `json:"parse_mode,omitempty"` // "HTML", "Markdown", "MarkdownV2"
	ReplyMarkup     map[string]interface{} `json:"reply_markup,omitempty"`
	MessageThreadID *int64                 `json:"message_thread_id,omitempty"` // ID топика форума
}

// ChatInfo информация о чате
type ChatInfo struct {
	ID int64 `json:"id"`
}

// SendMessageResult результат отправки сообщения
type SendMessageResult struct {
	MessageID int64    `json:"message_id"`
	Chat      ChatInfo `json:"chat"`
	Text      string   `json:"text"`
	Date      int64    `json:"date"`
}

// SendMessageResponse ответ от Telegram API
type SendMessageResponse struct {
	APIResponse
	Result SendMessageResult `json:"result"`
}

// SendMessage отправляет текстовое сообщение
func (c *Client) SendMessage(ctx context.Context, chatID int64, text string) error {
	_, err := c.SendMessageWithID(ctx, chatID, text)
	return err
}

// SendMessageWithID отправляет текстовое сообщение и возвращает messageID
func (c *Client) SendMessageWithID(ctx context.Context, chatID int64, text string) (int64, error) {
	req := SendMessageRequest{
		ChatID: chatID,
		Text:   text,
	}

	return c.sendMessage(ctx, req)
}

// SendMessageWithIDAndHTML отправляет текстовое сообщение с HTML форматированием и возвращает messageID
func (c *Client) SendMessageWithIDAndHTML(ctx context.Context, chatID int64, text string) (int64, error) {
	req := SendMessageRequest{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "HTML",
	}

	return c.sendMessage(ctx, req)
}

// SendMessageWithMarkdown отправляет текстовое сообщение с Markdown форматированием
func (c *Client) SendMessageWithMarkdown(ctx context.Context, chatID int64, text string) error {
	req := SendMessageRequest{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "Markdown",
	}

	_, err := c.sendMessage(ctx, req)
	return err
}

// SendMessageWithKeyboard отправляет сообщение с клавиатурой
func (c *Client) SendMessageWithKeyboard(ctx context.Context, chatID int64, text string, keyboard map[string]interface{}) error {
	req := SendMessageRequest{
		ChatID:      chatID,
		Text:        text,
		ReplyMarkup: keyboard,
	}

	_, err := c.sendMessage(ctx, req)
	return err
}

// SendMessageWithKeyboardAndMarkdown отправляет сообщение с клавиатурой и Markdown форматированием
func (c *Client) SendMessageWithKeyboardAndMarkdown(ctx context.Context, chatID int64, text string, keyboard map[string]interface{}) error {
	req := SendMessageRequest{
		ChatID:      chatID,
		Text:        text,
		ParseMode:   "Markdown",
		ReplyMarkup: keyboard,
	}

	_, err := c.sendMessage(ctx, req)
	return err
}

// SendMessageWithRequest отправляет сообщение с кастомным запросом (для поддержки message_thread_id и других параметров)
func (c *Client) SendMessageWithRequest(ctx context.Context, req SendMessageRequest) (int64, error) {
	return c.sendMessage(ctx, req)
}

// sendMessage выполняет запрос к Telegram API для отправки сообщения и возвращает messageID
func (c *Client) sendMessage(ctx context.Context, req SendMessageRequest) (int64, error) {
	url := c.baseURL + "/sendMessage"

	jsonData, err := json.Marshal(req)
	if err != nil {
		// Системная ошибка - Error (критично)
		c.log.Error("failed to marshal request",
			"error", err,
			"chat_id", req.ChatID,
		)
		return 0, fmt.Errorf("telegram marshal failed [chat_id=%d]: %w", req.ChatID, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		// Системная ошибка - Error (критично)
		c.log.Error("failed to create request",
			"error", err,
			"chat_id", req.ChatID,
		)
		return 0, fmt.Errorf("telegram create request failed [chat_id=%d]: %w", req.ChatID, err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		// Ошибка соединения - Debug
		c.log.Debug("telegram request failed",
			"error", err,
			"chat_id", req.ChatID,
		)
		return 0, fmt.Errorf("telegram request failed [chat_id=%d]: %w", req.ChatID, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// Системная ошибка - Error (критично)
		c.log.Error("failed to read response body",
			"error", err,
			"chat_id", req.ChatID,
			"status_code", resp.StatusCode,
		)
		return 0, fmt.Errorf("telegram read body failed [chat_id=%d, status=%d]: %w",
			req.ChatID, resp.StatusCode, err)
	}

	var apiResp SendMessageResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		// Системная ошибка - Error (критично)
		c.log.Error("failed to unmarshal response",
			"error", err,
			"chat_id", req.ChatID,
			"status_code", resp.StatusCode,
			"body_preview", truncateString(string(body), 200),
		)
		return 0, fmt.Errorf("telegram unmarshal failed [chat_id=%d, status=%d]: %w",
			req.ChatID, resp.StatusCode, err)
	}

	if !apiResp.OK {
		// Ошибка внешнего API - Debug
		c.log.Debug("telegram API error",
			"error_code", apiResp.ErrorCode,
			"description", apiResp.Description,
			"chat_id", req.ChatID,
			"status_code", resp.StatusCode,
		)
		return 0, fmt.Errorf("telegram API error [code=%d, chat_id=%d]: %s",
			apiResp.ErrorCode, req.ChatID, apiResp.Description)
	}

	c.log.Debug("message sent successfully",
		"chat_id", req.ChatID,
		"message_id", apiResp.Result.MessageID,
	)

	return apiResp.Result.MessageID, nil
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
		c.log.Debug("getMe failed",
			"status_code", resp.StatusCode,
			"body", string(body),
		)
		return fmt.Errorf("getMe failed [status=%d]: %s", resp.StatusCode, string(body))
	}

	c.log.Debug("bot info retrieved successfully")
	return nil
}

// BotCommand представляет команду бота
type BotCommand struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

// BotCommandScope представляет область действия команд
type BotCommandScope struct {
	Type string `json:"type"`
}

// SetMyCommandsRequest запрос на установку команд бота
type SetMyCommandsRequest struct {
	Commands []BotCommand    `json:"commands"`
	Scope    BotCommandScope `json:"scope"`
}

// SetMyCommands регистрирует команды бота в меню
func (c *Client) SetMyCommands(ctx context.Context, commands []BotCommand) error {
	reqBody := SetMyCommandsRequest{
		Commands: commands,
		Scope: BotCommandScope{
			Type: "all_private_chats", // Команды только для приватных чатов
		},
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
		// Системная ошибка - Error (критично)
		c.log.Error("failed to unmarshal response",
			"error", err,
			"status_code", resp.StatusCode,
			"body_preview", truncateString(string(body), 200),
		)
		return fmt.Errorf("telegram unmarshal failed [status=%d]: %w", resp.StatusCode, err)
	}

	if !apiResp.OK {
		// Ошибка внешнего API - Debug
		c.log.Debug("telegram API error",
			"error_code", apiResp.ErrorCode,
			"description", apiResp.Description,
			"status_code", resp.StatusCode,
		)
		return fmt.Errorf("telegram API error [code=%d]: %s", apiResp.ErrorCode, apiResp.Description)
	}

	c.log.Debug("bot commands registered successfully", "commands_count", len(commands))
	return nil
}

// SetWebhookRequest запрос на установку webhook
type SetWebhookRequest struct {
	URL         string `json:"url"`
	SecretToken string `json:"secret_token,omitempty"`
}

// SetWebhook устанавливает webhook для бота
// url - URL для получения обновлений
// secretToken - domain.BotId (будет отправлен в заголовке X-Telegram-Bot-Api-Secret-Token)
func (c *Client) SetWebhook(ctx context.Context, url string, secretToken string) error {
	reqBody := SetWebhookRequest{
		URL:         url,
		SecretToken: secretToken,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	apiURL := c.baseURL + "/setWebhook"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(jsonData))
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
		// Системная ошибка - Error (критично)
		c.log.Error("failed to unmarshal response",
			"error", err,
			"status_code", resp.StatusCode,
			"body_preview", truncateString(string(body), 200),
		)
		return fmt.Errorf("telegram unmarshal failed [status=%d]: %w", resp.StatusCode, err)
	}

	if !apiResp.OK {
		// Ошибка внешнего API - Debug
		c.log.Debug("telegram API error",
			"error_code", apiResp.ErrorCode,
			"description", apiResp.Description,
			"status_code", resp.StatusCode,
			"url", url,
		)
		return fmt.Errorf("telegram API error [code=%d, url=%s]: %s",
			apiResp.ErrorCode, url, apiResp.Description)
	}

	c.log.Debug("webhook set successfully",
		"url", url,
		"has_secret_token", secretToken != "",
	)
	return nil
}

// EditMessageReplyMarkupRequest запрос на редактирование reply_markup сообщения
type EditMessageReplyMarkupRequest struct {
	ChatID      int64                  `json:"chat_id"`
	MessageID   int64                  `json:"message_id"`
	ReplyMarkup map[string]interface{} `json:"reply_markup,omitempty"`
}

// EditMessageReplyMarkup редактирует reply_markup сообщения (убирает кнопки, если передать пустой reply_markup)
func (c *Client) EditMessageReplyMarkup(ctx context.Context, chatID int64, messageID int64, replyMarkup map[string]interface{}) error {
	reqBody := EditMessageReplyMarkupRequest{
		ChatID:      chatID,
		MessageID:   messageID,
		ReplyMarkup: replyMarkup,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		c.log.Error("failed to marshal edit message reply markup request",
			"error", err,
			"chat_id", chatID,
			"message_id", messageID,
		)
		return fmt.Errorf("telegram marshal failed [chat_id=%d, message_id=%d]: %w", chatID, messageID, err)
	}

	url := c.baseURL + "/editMessageReplyMarkup"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		c.log.Error("failed to create edit message reply markup request",
			"error", err,
			"chat_id", chatID,
			"message_id", messageID,
		)
		return fmt.Errorf("telegram create request failed [chat_id=%d, message_id=%d]: %w", chatID, messageID, err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.log.Debug("telegram edit message reply markup request failed",
			"error", err,
			"chat_id", chatID,
			"message_id", messageID,
		)
		return fmt.Errorf("telegram request failed [chat_id=%d, message_id=%d]: %w", chatID, messageID, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.log.Error("failed to read edit message reply markup response body",
			"error", err,
			"chat_id", chatID,
			"message_id", messageID,
			"status_code", resp.StatusCode,
		)
		return fmt.Errorf("telegram read body failed [chat_id=%d, message_id=%d, status=%d]: %w",
			chatID, messageID, resp.StatusCode, err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		c.log.Error("failed to unmarshal edit message reply markup response",
			"error", err,
			"chat_id", chatID,
			"message_id", messageID,
			"status_code", resp.StatusCode,
			"body_preview", truncateString(string(body), 200),
		)
		return fmt.Errorf("telegram unmarshal failed [chat_id=%d, message_id=%d, status=%d]: %w",
			chatID, messageID, resp.StatusCode, err)
	}

	if !apiResp.OK {
		c.log.Debug("telegram API error on edit message reply markup",
			"error_code", apiResp.ErrorCode,
			"description", apiResp.Description,
			"chat_id", chatID,
			"message_id", messageID,
			"status_code", resp.StatusCode,
		)
		return fmt.Errorf("telegram API error [code=%d, chat_id=%d, message_id=%d]: %s",
			apiResp.ErrorCode, chatID, messageID, apiResp.Description)
	}

	c.log.Debug("message reply markup edited successfully",
		"chat_id", chatID,
		"message_id", messageID,
	)
	return nil
}

// DeleteMessageRequest запрос на удаление сообщения
type DeleteMessageRequest struct {
	ChatID    int64 `json:"chat_id"`
	MessageID int64 `json:"message_id"`
}

// DeleteMessage удаляет сообщение из чата
func (c *Client) DeleteMessage(ctx context.Context, chatID int64, messageID int64) error {
	reqBody := DeleteMessageRequest{
		ChatID:    chatID,
		MessageID: messageID,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		c.log.Error("failed to marshal delete message request",
			"error", err,
			"chat_id", chatID,
			"message_id", messageID,
		)
		return fmt.Errorf("telegram marshal failed [chat_id=%d, message_id=%d]: %w", chatID, messageID, err)
	}

	url := c.baseURL + "/deleteMessage"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		c.log.Error("failed to create delete message request",
			"error", err,
			"chat_id", chatID,
			"message_id", messageID,
		)
		return fmt.Errorf("telegram create request failed [chat_id=%d, message_id=%d]: %w", chatID, messageID, err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.log.Debug("telegram delete message request failed",
			"error", err,
			"chat_id", chatID,
			"message_id", messageID,
		)
		return fmt.Errorf("telegram request failed [chat_id=%d, message_id=%d]: %w", chatID, messageID, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.log.Error("failed to read delete message response body",
			"error", err,
			"chat_id", chatID,
			"message_id", messageID,
			"status_code", resp.StatusCode,
		)
		return fmt.Errorf("telegram read body failed [chat_id=%d, message_id=%d, status=%d]: %w",
			chatID, messageID, resp.StatusCode, err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		c.log.Error("failed to unmarshal delete message response",
			"error", err,
			"chat_id", chatID,
			"message_id", messageID,
			"status_code", resp.StatusCode,
			"body_preview", truncateString(string(body), 200),
		)
		return fmt.Errorf("telegram unmarshal failed [chat_id=%d, message_id=%d, status=%d]: %w",
			chatID, messageID, resp.StatusCode, err)
	}

	if !apiResp.OK {
		// Если сообщение уже удалено или не найдено - это не критичная ошибка
		if apiResp.ErrorCode == 400 && (apiResp.Description == "Bad Request: message to delete not found" ||
			apiResp.Description == "Bad Request: message can't be deleted") {
			c.log.Debug("message already deleted or can't be deleted",
				"chat_id", chatID,
				"message_id", messageID,
			)
			return nil // Не возвращаем ошибку, если сообщение уже удалено
		}
		c.log.Debug("telegram API error on delete message",
			"error_code", apiResp.ErrorCode,
			"description", apiResp.Description,
			"chat_id", chatID,
			"message_id", messageID,
			"status_code", resp.StatusCode,
		)
		return fmt.Errorf("telegram API error [code=%d, chat_id=%d, message_id=%d]: %s",
			apiResp.ErrorCode, chatID, messageID, apiResp.Description)
	}

	c.log.Debug("message deleted successfully",
		"chat_id", chatID,
		"message_id", messageID,
	)
	return nil
}
