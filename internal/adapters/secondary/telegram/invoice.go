package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// LabeledPrice представляет цену в invoice
type LabeledPrice struct {
	Label  string `json:"label"`  // название позиции
	Amount int64  `json:"amount"` // цена в минимальных единицах валюты (для Stars - количество звёзд)
}

// SendInvoiceRequest запрос на отправку invoice (для Telegram Stars)
// Документация: https://core.telegram.org/bots/api#sendinvoice
type SendInvoiceRequest struct {
	ChatID                    int64          `json:"chat_id"`
	Title                     string         `json:"title"`                           // название продукта
	Description               string         `json:"description"`                     // описание продукта
	Payload                   string         `json:"payload"`                         // уникальный payload для идентификации платежа
	ProviderToken             string         `json:"provider_token,omitempty"`        // для внешних провайдеров (не нужен для Stars)
	Currency                  string         `json:"currency"`                        // "XTR" для Stars
	Prices                    []LabeledPrice `json:"prices"`                          // массив цен
	MaxTipAmount              *int64         `json:"max_tip_amount,omitempty"`        // максимальная сумма чаевых
	SuggestedTipAmounts       []int64        `json:"suggested_tip_amounts,omitempty"` // предложенные суммы чаевых
	StartParameter            *string        `json:"start_parameter,omitempty"`       // для deep linking
	ProviderData              *string        `json:"provider_data,omitempty"`         // данные провайдера
	PhotoURL                  *string        `json:"photo_url,omitempty"`
	PhotoSize                 *int64         `json:"photo_size,omitempty"`
	PhotoWidth                *int64         `json:"photo_width,omitempty"`
	PhotoHeight               *int64         `json:"photo_height,omitempty"`
	NeedName                  bool           `json:"need_name,omitempty"`
	NeedPhoneNumber           bool           `json:"need_phone_number,omitempty"`
	NeedEmail                 bool           `json:"need_email,omitempty"`
	NeedShippingAddress       bool           `json:"need_shipping_address,omitempty"`
	SendPhoneNumberToProvider bool           `json:"send_phone_number_to_provider,omitempty"`
	SendEmailToProvider       bool           `json:"send_email_to_provider,omitempty"`
	IsFlexible                bool           `json:"is_flexible,omitempty"`
	MessageThreadID           *int64         `json:"message_thread_id,omitempty"` // ID топика форума
}

// SendInvoiceResult результат отправки invoice
type SendInvoiceResult struct {
	MessageID int64    `json:"message_id"`
	Chat      ChatInfo `json:"chat"`
}

// SendInvoiceResponse ответ от Telegram API на sendInvoice
type SendInvoiceResponse struct {
	APIResponse
	Result SendInvoiceResult `json:"result"`
}

// SendInvoice отправляет invoice пользователю (для Telegram Stars)
func (c *Client) SendInvoice(ctx context.Context, req SendInvoiceRequest) (int64, error) {
	url := c.baseURL + "/sendInvoice"

	jsonData, err := json.Marshal(req)
	if err != nil {
		c.log.Error("failed to marshal sendInvoice request",
			"error", err,
			"chat_id", req.ChatID,
		)
		return 0, fmt.Errorf("telegram marshal failed [chat_id=%d]: %w", req.ChatID, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		c.log.Error("failed to create sendInvoice request",
			"error", err,
			"chat_id", req.ChatID,
		)
		return 0, fmt.Errorf("telegram create request failed [chat_id=%d]: %w", req.ChatID, err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.log.Debug("telegram sendInvoice request failed",
			"error", err,
			"chat_id", req.ChatID,
		)
		return 0, fmt.Errorf("telegram request failed [chat_id=%d]: %w", req.ChatID, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.log.Error("failed to read sendInvoice response body",
			"error", err,
			"chat_id", req.ChatID,
			"status_code", resp.StatusCode,
		)
		return 0, fmt.Errorf("telegram read body failed [chat_id=%d, status=%d]: %w",
			req.ChatID, resp.StatusCode, err)
	}

	var apiResp SendInvoiceResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		c.log.Error("failed to unmarshal sendInvoice response",
			"error", err,
			"chat_id", req.ChatID,
			"status_code", resp.StatusCode,
			"body_preview", truncateString(string(body), 200),
		)
		return 0, fmt.Errorf("telegram unmarshal failed [chat_id=%d, status=%d]: %w",
			req.ChatID, resp.StatusCode, err)
	}

	if !apiResp.OK {
		c.log.Debug("telegram sendInvoice API error",
			"error_code", apiResp.ErrorCode,
			"description", apiResp.Description,
			"chat_id", req.ChatID,
			"status_code", resp.StatusCode,
		)
		return 0, fmt.Errorf("telegram API error [code=%d, chat_id=%d]: %s",
			apiResp.ErrorCode, req.ChatID, apiResp.Description)
	}

	c.log.Debug("invoice sent successfully",
		"chat_id", req.ChatID,
		"message_id", apiResp.Result.MessageID,
	)

	return apiResp.Result.MessageID, nil
}

// AnswerPreCheckoutQueryRequest запрос на ответ pre_checkout_query
type AnswerPreCheckoutQueryRequest struct {
	PreCheckoutQueryID string  `json:"pre_checkout_query_id"`
	OK                 bool    `json:"ok"`                      // true - подтвердить, false - отклонить
	ErrorMessage       *string `json:"error_message,omitempty"` // сообщение об ошибке (если ok=false)
}

// AnswerPreCheckoutQuery отвечает на pre_checkout_query (подтверждает или отклоняет платёж)
// Документация: https://core.telegram.org/bots/api#answerprecheckoutquery
func (c *Client) AnswerPreCheckoutQuery(ctx context.Context, queryID string, ok bool, errorMessage *string) error {
	reqBody := AnswerPreCheckoutQueryRequest{
		PreCheckoutQueryID: queryID,
		OK:                 ok,
		ErrorMessage:       errorMessage,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		c.log.Error("failed to marshal answerPreCheckoutQuery request",
			"error", err,
			"query_id", queryID,
		)
		return fmt.Errorf("telegram marshal failed [query_id=%s]: %w", queryID, err)
	}

	url := c.baseURL + "/answerPreCheckoutQuery"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		c.log.Error("failed to create answerPreCheckoutQuery request",
			"error", err,
			"query_id", queryID,
		)
		return fmt.Errorf("telegram create request failed [query_id=%s]: %w", queryID, err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.log.Debug("telegram answerPreCheckoutQuery request failed",
			"error", err,
			"query_id", queryID,
		)
		return fmt.Errorf("telegram request failed [query_id=%s]: %w", queryID, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.log.Error("failed to read answerPreCheckoutQuery response body",
			"error", err,
			"query_id", queryID,
			"status_code", resp.StatusCode,
		)
		return fmt.Errorf("telegram read body failed [query_id=%s, status=%d]: %w",
			queryID, resp.StatusCode, err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		c.log.Error("failed to unmarshal answerPreCheckoutQuery response",
			"error", err,
			"query_id", queryID,
			"status_code", resp.StatusCode,
			"body_preview", truncateString(string(body), 200),
		)
		return fmt.Errorf("telegram unmarshal failed [query_id=%s, status=%d]: %w",
			queryID, resp.StatusCode, err)
	}

	if !apiResp.OK {
		c.log.Debug("telegram answerPreCheckoutQuery API error",
			"error_code", apiResp.ErrorCode,
			"description", apiResp.Description,
			"query_id", queryID,
			"status_code", resp.StatusCode,
		)
		return fmt.Errorf("telegram API error [code=%d, query_id=%s]: %s",
			apiResp.ErrorCode, queryID, apiResp.Description)
	}

	c.log.Debug("pre_checkout_query answered successfully",
		"query_id", queryID,
		"ok", ok,
	)
	return nil
}
