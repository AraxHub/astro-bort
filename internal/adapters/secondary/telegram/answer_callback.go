package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// AnswerCallbackQueryRequest запрос на ответ callback query
type AnswerCallbackQueryRequest struct {
	CallbackQueryID string `json:"callback_query_id"`
	Text            string `json:"text,omitempty"`
	ShowAlert       bool   `json:"show_alert,omitempty"`
}

// AnswerCallbackQuery отправляет ответ на callback query
func (c *Client) AnswerCallbackQuery(ctx context.Context, callbackID string, text string, showAlert bool) error {
	reqBody := AnswerCallbackQueryRequest{
		CallbackQueryID: callbackID,
		Text:            text,
		ShowAlert:       showAlert,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + "/answerCallbackQuery"
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

	c.log.Debug("callback query answered successfully", "callback_id", callbackID)
	return nil
}
