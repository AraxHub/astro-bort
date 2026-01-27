package astro

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/google/uuid"
)

const (
	maxMessageLength      = 4000  // максимальная длина сообщения Telegram
	chunkSize             = 3900  // размер части для разбиения
	messageDelaySeconds   = 1     // задержка между отправками в секундах
)

// HandleRAGResponse обрабатывает ответ от RAG
func (s *Service) HandleRAGResponse(ctx context.Context, requestID uuid.UUID, botID domain.BotId, chatID int64, responseText string) (err error) {
	var statusStage domain.RequestStage
	var statusErrorCode string
	var statusMetadata json.RawMessage

	defer func() {
		if err != nil {
			// Ошибка - создаём статус ошибки
			errMsg := err.Error()
			metadata := domain.BuildErrorMetadata(
				statusStage,
				statusErrorCode,
				string(botID),
				map[string]interface{}{
					"request_id": requestID.String(),
					"chat_id":    chatID,
				},
			)

			status := &domain.Status{
				ID:           uuid.New(),
				ObjectType:   domain.ObjectTypeRequest,
				ObjectID:     requestID,
				Status:       domain.StatusStatus(domain.RequestError),
				ErrorMessage: &errMsg,
				Metadata:     metadata,
				CreatedAt:    time.Now(),
			}
			s.createOrLogStatus(ctx, status)
			s.sendAlertOrLog(ctx, status)
		} else {
			// Успех - создаём финальный статус (алерт не отправляем для успешных кейсов)
			if statusMetadata == nil {
				// Если metadata не был установлен, не создаём статус
				return
			}

			status := &domain.Status{
				ID:         uuid.New(),
				ObjectType: domain.ObjectTypeRequest,
				ObjectID:   requestID,
				Status:     domain.StatusStatus(domain.RequestCompleted),
				Metadata:   statusMetadata,
				CreatedAt:  time.Now(),
			}
			s.createOrLogStatus(ctx, status)
		}
	}()

	// Сохраняем ответ напрямую по request_id (без SELECT)
	if err = s.RequestRepo.UpdateResponseTextByID(ctx, requestID, responseText); err != nil {
		statusStage = domain.StageSaveResponse
		statusErrorCode = "DB_UPDATE_ERROR"
		s.Log.Error("failed to update request with response",
			"error", err,
			"request_id", requestID,
		)
		return fmt.Errorf("failed to update request: %w", err)
	}

	// Отправляем сообщение с bot_id и chat_id из Kafka (без SELECT User)
	// Если сообщение длиннее maxMessageLength, разбиваем на части
	var firstMessageID int64
	var totalParts int

	if len([]rune(responseText)) <= maxMessageLength {
		// Короткое сообщение - отправляем как есть с HTML форматированием
		messageID, err := s.TelegramService.SendMessageWithIDAndHTML(ctx, botID, chatID, responseText)
		if err != nil {
			statusStage = domain.StageSendTelegram
			statusErrorCode = "TELEGRAM_SEND_ERROR"
			if strings.Contains(err.Error(), "429") {
				statusErrorCode = "TELEGRAM_RATE_LIMIT"
			} else if strings.Contains(err.Error(), "timeout") {
				statusErrorCode = "TELEGRAM_TIMEOUT"
			}
			s.Log.Error("failed to send RAG response to user",
				"error", err,
				"request_id", requestID,
				"bot_id", botID,
				"chat_id", chatID,
			)
			return fmt.Errorf("failed to send response: %w", err)
		}
		firstMessageID = messageID
		totalParts = 1
	} else {
		// Длинное сообщение - разбиваем на части
		parts := splitLongMessage(responseText, chunkSize)
		totalParts = len(parts)

		s.Log.Info("splitting long message into parts",
			"request_id", requestID,
			"total_length", len([]rune(responseText)),
			"parts_count", totalParts,
		)

		for i, part := range parts {
			// Задержка перед отправкой каждой части (кроме первой)
			if i > 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(time.Duration(messageDelaySeconds) * time.Second):
				}
			}

			messageID, err := s.TelegramService.SendMessageWithIDAndHTML(ctx, botID, chatID, part)
			if err != nil {
				statusStage = domain.StageSendTelegram
				statusErrorCode = "TELEGRAM_SEND_ERROR"
				if strings.Contains(err.Error(), "429") {
					statusErrorCode = "TELEGRAM_RATE_LIMIT"
				} else if strings.Contains(err.Error(), "timeout") {
					statusErrorCode = "TELEGRAM_TIMEOUT"
				}
				s.Log.Error("failed to send RAG response part to user",
					"error", err,
					"request_id", requestID,
					"bot_id", botID,
					"chat_id", chatID,
					"part", i+1,
					"total_parts", totalParts,
				)
				return fmt.Errorf("failed to send response part %d/%d: %w", i+1, totalParts, err)
			}

			// Сохраняем ID первого сообщения
			if i == 0 {
				firstMessageID = messageID
			}

			s.Log.Debug("sent message part",
				"request_id", requestID,
				"part", i+1,
				"total_parts", totalParts,
				"part_length", len([]rune(part)),
			)
		}
	}

	// Успех - формируем metadata
	statusMetadata = domain.BuildTelegramMetadata(
		firstMessageID,
		chatID,
		string(botID),
		len(responseText),
	)

	s.Log.Info("RAG response sent to user",
		"request_id", requestID,
		"message_id", firstMessageID,
		"bot_id", botID,
		"chat_id", chatID,
		"total_parts", totalParts,
	)

	return nil
}

// splitLongMessage разбивает длинное сообщение на части по chunkSize символов,
// стараясь не резать по середине слова или предложения
func splitLongMessage(text string, chunkSize int) []string {
	runes := []rune(text)
	if len(runes) <= chunkSize {
		return []string{text}
	}

	var parts []string
	start := 0

	for start < len(runes) {
		end := start + chunkSize
		if end > len(runes) {
			end = len(runes)
		}

		// Если это не последняя часть, пытаемся найти хорошее место для разрыва
		if end < len(runes) {
			// Ищем конец предложения (точка, восклицательный знак, вопросительный знак)
			bestBreak := findSentenceEnd(runes, start, end)
			if bestBreak > start {
				end = bestBreak + 1
			} else {
				// Если не нашли конец предложения, ищем пробел или знак препинания
				bestBreak = findWordBreak(runes, start, end)
				if bestBreak > start {
					end = bestBreak + 1
				}
			}
		}

		parts = append(parts, string(runes[start:end]))
		start = end
	}

	return parts
}

// findSentenceEnd ищет конец предложения в диапазоне [start, end)
// Возвращает позицию последнего найденного знака конца предложения
func findSentenceEnd(runes []rune, start, end int) int {
	sentenceEnders := []rune{'.', '!', '?', '。', '！', '？'}
	bestBreak := -1

	// Ищем с конца диапазона, чтобы найти последний конец предложения
	for i := end - 1; i >= start && i >= end-200; i-- { // проверяем последние 200 символов
		for _, se := range sentenceEnders {
			if runes[i] == se {
				// Проверяем, что после знака препинания идет пробел или конец текста
				if i+1 >= len(runes) || unicode.IsSpace(runes[i+1]) {
					if i > bestBreak {
						bestBreak = i
					}
					break
				}
			}
		}
	}

	return bestBreak
}

// findWordBreak ищет место разрыва по границе слова (пробел или знак препинания)
// Возвращает позицию последнего найденного пробела/знака препинания
func findWordBreak(runes []rune, start, end int) int {
	bestBreak := -1

	// Ищем с конца диапазона, чтобы не резать слово
	for i := end - 1; i >= start && i >= end-100; i-- { // проверяем последние 100 символов
		isSpace := unicode.IsSpace(runes[i])
		isPunct := unicode.IsPunct(runes[i]) && runes[i] != '-' && runes[i] != '_'
		
		if (isSpace || isPunct) && i > bestBreak {
			bestBreak = i
		}
	}

	return bestBreak
}
