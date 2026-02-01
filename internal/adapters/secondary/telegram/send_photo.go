package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

// SendPhotoRequest запрос на отправку фото
type SendPhotoRequest struct {
	ChatID          int64
	Photo           []byte
	Filename        string
	Caption         string
	MessageThreadID *int64
}

// PhotoSize размер фото
type PhotoSize struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	FileSize     *int   `json:"file_size,omitempty"`
}

// MessagePhoto сообщение с фото
type MessagePhoto struct {
	MessageID int64       `json:"message_id"`
	Chat      ChatInfo    `json:"chat"`
	Photo     []PhotoSize `json:"photo"`
	Date      int64       `json:"date"`
}

// SendPhotoResult результат отправки фото
type SendPhotoResult struct {
	MessageID int64       `json:"message_id"`
	Chat      ChatInfo    `json:"chat"`
	Photo     []PhotoSize `json:"photo"`
	Date      int64       `json:"date"`
}

// SendPhotoResponse ответ от Telegram API на sendPhoto
type SendPhotoResponse struct {
	APIResponse
	Result SendPhotoResult `json:"result"`
}

// SendPhoto отправляет фото в чат и возвращает file_id
func (c *Client) SendPhoto(ctx context.Context, chatID int64, messageThreadID *int64, photoData []byte, filename string) (string, error) {
	// Создаём multipart/form-data запрос
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Добавляем chat_id
	if err := writer.WriteField("chat_id", fmt.Sprintf("%d", chatID)); err != nil {
		c.log.Error("failed to write chat_id field",
			"error", err,
			"chat_id", chatID)
		return "", fmt.Errorf("failed to write chat_id: %w", err)
	}

	// Добавляем message_thread_id если указан
	if messageThreadID != nil {
		if err := writer.WriteField("message_thread_id", fmt.Sprintf("%d", *messageThreadID)); err != nil {
			c.log.Error("failed to write message_thread_id field",
				"error", err,
				"message_thread_id", *messageThreadID)
			return "", fmt.Errorf("failed to write message_thread_id: %w", err)
		}
	}

	// Добавляем фото как файл
	photoPart, err := writer.CreateFormFile("photo", filename)
	if err != nil {
		c.log.Error("failed to create photo form file",
			"error", err,
			"filename", filename)
		return "", fmt.Errorf("failed to create photo form file: %w", err)
	}

	if _, err := photoPart.Write(photoData); err != nil {
		c.log.Error("failed to write photo data",
			"error", err,
			"filename", filename)
		return "", fmt.Errorf("failed to write photo data: %w", err)
	}

	// Закрываем writer для завершения multipart
	if err := writer.Close(); err != nil {
		c.log.Error("failed to close multipart writer",
			"error", err)
		return "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Создаём HTTP запрос
	url := c.baseURL + "/sendPhoto"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &requestBody)
	if err != nil {
		c.log.Error("failed to create sendPhoto request",
			"error", err,
			"chat_id", chatID,
			"message_thread_id", messageThreadID)
		return "", fmt.Errorf("telegram create request failed [chat_id=%d]: %w", chatID, err)
	}

	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	c.log.Debug("sending photo to Telegram",
		"chat_id", chatID,
		"message_thread_id", messageThreadID,
		"filename", filename,
		"photo_size", len(photoData),
		"url", url)

	// Отправляем запрос
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.log.Debug("telegram sendPhoto request failed",
			"error", err,
			"chat_id", chatID,
			"filename", filename)
		return "", fmt.Errorf("telegram request failed [chat_id=%d]: %w", chatID, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.log.Error("failed to read sendPhoto response body",
			"error", err,
			"chat_id", chatID,
			"status_code", resp.StatusCode)
		return "", fmt.Errorf("telegram read response failed [chat_id=%d, status=%d]: %w",
			chatID, resp.StatusCode, err)
	}

	var apiResp SendPhotoResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		c.log.Error("failed to unmarshal sendPhoto response",
			"error", err,
			"chat_id", chatID,
			"status_code", resp.StatusCode,
			"body_preview", truncateString(string(body), 200),
		)
		return "", fmt.Errorf("telegram unmarshal failed [chat_id=%d, status=%d]: %w",
			chatID, resp.StatusCode, err)
	}

	if !apiResp.OK {
		c.log.Debug("telegram API error",
			"error_code", apiResp.ErrorCode,
			"description", apiResp.Description,
			"chat_id", chatID,
			"status_code", resp.StatusCode,
		)
		return "", fmt.Errorf("telegram API error [code=%d, chat_id=%d]: %s",
			apiResp.ErrorCode, chatID, apiResp.Description)
	}

	// Извлекаем file_id из самого большого размера фото (обычно последний в массиве)
	if len(apiResp.Result.Photo) == 0 {
		return "", fmt.Errorf("no photo sizes in response")
	}

	// Берём последний элемент (обычно самый большой размер)
	largestPhoto := apiResp.Result.Photo[len(apiResp.Result.Photo)-1]
	fileID := largestPhoto.FileID

	c.log.Debug("photo sent successfully",
		"chat_id", chatID,
		"message_id", apiResp.Result.MessageID,
		"file_id", fileID,
		"filename", filename)

	return fileID, nil
}

// SendPhotoByFileID отправляет фото в чат используя уже существующий file_id
func (c *Client) SendPhotoByFileID(ctx context.Context, chatID int64, fileID string) error {
	reqBody := map[string]interface{}{
		"chat_id": chatID,
		"photo":   fileID,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		c.log.Error("failed to marshal sendPhotoByFileID request",
			"error", err,
			"chat_id", chatID)
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + "/sendPhoto"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		c.log.Error("failed to create sendPhotoByFileID request",
			"error", err,
			"chat_id", chatID)
		return fmt.Errorf("telegram create request failed [chat_id=%d]: %w", chatID, err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.log.Error("telegram sendPhotoByFileID request failed",
			"error", err,
			"chat_id", chatID,
			"file_id", fileID)
		return fmt.Errorf("telegram request failed [chat_id=%d]: %w", chatID, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.log.Warn("failed to read sendPhotoByFileID response body",
			"error", err,
			"chat_id", chatID,
			"status_code", resp.StatusCode)
		return fmt.Errorf("telegram read response failed [chat_id=%d, status=%d]: %w",
			chatID, resp.StatusCode, err)
	}

	var apiResp SendPhotoResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		c.log.Error("failed to unmarshal sendPhotoByFileID response",
			"error", err,
			"chat_id", chatID,
			"status_code", resp.StatusCode,
			"body_preview", truncateString(string(body), 200),
		)
		return fmt.Errorf("telegram unmarshal failed [chat_id=%d, status=%d]: %w",
			chatID, resp.StatusCode, err)
	}

	if !apiResp.OK {
		c.log.Error("telegram API error",
			"error_code", apiResp.ErrorCode,
			"description", apiResp.Description,
			"chat_id", chatID,
			"status_code", resp.StatusCode,
		)
		return fmt.Errorf("telegram API error [code=%d, chat_id=%d]: %s",
			apiResp.ErrorCode, chatID, apiResp.Description)
	}

	return nil
}
