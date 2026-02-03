package astro

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/google/uuid"
)

// SendImageForTheme отправляет картинку по теме для указанного запроса
func (s *Service) SendImageForTheme(ctx context.Context, requestID uuid.UUID, botID domain.BotId, chatID int64, themeStr string) (err error) {
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
					"theme":      themeStr,
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
	if s.ImageRepo == nil || s.ImageUsageRepo == nil {
		return fmt.Errorf("image repositories not configured")
	}

	if s.TelegramService == nil {
		return fmt.Errorf("telegram service not configured")
	}

	theme := domain.ImageTheme(themeStr)
	if !theme.IsValid() {
		return fmt.Errorf("invalid theme: %s", themeStr)
	}

	// Получаем image_usage для чата
	statusStage = domain.StageGetImageUsage
	usage, err := s.ImageUsageRepo.GetUsage(ctx, chatID)
	if err != nil {
		// Проверяем, что ошибка именно "не найдено", а не ошибка парсинга или другая
		if !errors.Is(err, sql.ErrNoRows) {
			statusErrorCode = "DB_GET_IMAGE_USAGE_ERROR"
			s.Log.Error("failed to get image usage",
				"error", err,
				"request_id", requestID,
				"chat_id", chatID)
			return fmt.Errorf("failed to get image usage: %w", err)
		}
		// Если не найдено - создаём новую запись
		now := time.Now()
		usage = &domain.ImageUsage{
			ChatID:     chatID,
			UsedImages: make(map[string]int),
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if err := s.ImageUsageRepo.Create(ctx, usage); err != nil {
			statusErrorCode = "DB_CREATE_IMAGE_USAGE_ERROR"
			s.Log.Error("failed to create image usage",
				"error", err,
				"request_id", requestID,
				"chat_id", chatID)
			return fmt.Errorf("failed to create image usage: %w", err)
		}
	}

	// Получаем все картинки по теме
	statusStage = domain.StageGetImages
	images, err := s.ImageRepo.GetByTheme(ctx, theme)
	if err != nil {
		statusErrorCode = "DB_GET_IMAGES_ERROR"
		s.Log.Error("failed to get images by theme",
			"error", err,
			"request_id", requestID,
			"theme", theme)
		return fmt.Errorf("failed to get images by theme: %w", err)
	}

	if len(images) == 0 {
		statusErrorCode = "NO_IMAGES_FOUND"
		s.Log.Warn("no images found for theme",
			"theme", theme,
			"request_id", requestID)
		return fmt.Errorf("no images found for theme: %s", theme)
	}

	// Lazy initialization: добавляем новые картинки в used_images с count=0
	needsUpdate := false
	for _, img := range images {
		if _, exists := usage.UsedImages[img.Filename]; !exists {
			usage.UsedImages[img.Filename] = 0
			needsUpdate = true
		}
	}

	// Обновляем usage в БД если были новые картинки
	if needsUpdate {
		if err := s.ImageUsageRepo.UpdateUsage(ctx, chatID, usage.UsedImages); err != nil {
			s.Log.Warn("failed to update image usage with new images",
				"error", err,
				"chat_id", chatID)
			// Продолжаем, это не критично
		}
	}

	// Находим минимальный count среди картинок темы
	minCount := -1
	for _, img := range images {
		count := usage.UsedImages[img.Filename]
		if minCount == -1 || count < minCount {
			minCount = count
		}
	}

	// Выбираем все картинки с минимальным count
	var candidates []*domain.Image
	for _, img := range images {
		if usage.UsedImages[img.Filename] == minCount {
			candidates = append(candidates, img)
		}
	}

	if len(candidates) == 0 {
		statusStage = domain.StageSelectImage
		statusErrorCode = "NO_CANDIDATES_FOUND"
		return fmt.Errorf("no candidates found for theme: %s", theme)
	}

	// Выбираем случайную картинку из кандидатов
	statusStage = domain.StageSelectImage
	var selectedImage *domain.Image
	if len(candidates) == 1 {
		selectedImage = candidates[0]
	} else {
		// Генерируем случайный индекс используя crypto/rand
		var randomBytes [8]byte
		if _, err := rand.Read(randomBytes[:]); err != nil {
			s.Log.Warn("failed to generate random number, using first candidate",
				"error", err,
				"candidates_count", len(candidates))
			selectedImage = candidates[0]
		} else {
			randomIndex := binary.BigEndian.Uint64(randomBytes[:]) % uint64(len(candidates))
			selectedImage = candidates[randomIndex]
		}
	}

	// Отправляем фото в Telegram используя file_id
	statusStage = domain.StageSendPhoto
	if selectedImage.TgFileID == "" {
		statusErrorCode = "NO_TELEGRAM_FILE_ID"
		return fmt.Errorf("image has no telegram file_id: %s", selectedImage.Filename)
	}

	// Отправляем фото через TelegramService используя file_id
	err = s.TelegramService.SendPhotoByFileID(ctx, botID, chatID, selectedImage.TgFileID)
	if err != nil {
		statusErrorCode = "TELEGRAM_SEND_ERROR"
		if strings.Contains(err.Error(), "429") {
			statusErrorCode = "TELEGRAM_RATE_LIMIT"
		} else if strings.Contains(err.Error(), "timeout") {
			statusErrorCode = "TELEGRAM_TIMEOUT"
		}
		s.Log.Error("failed to send photo",
			"error", err,
			"request_id", requestID,
			"bot_id", botID,
			"chat_id", chatID,
			"filename", selectedImage.Filename)
		return fmt.Errorf("failed to send photo: %w", err)
	}

	// Инкрементируем счётчик использования
	statusStage = domain.StageIncrementUsage
	if err := s.ImageUsageRepo.IncrementUsage(ctx, chatID, selectedImage.Filename); err != nil {
		s.Log.Warn("failed to increment image usage",
			"error", err,
			"chat_id", chatID,
			"filename", selectedImage.Filename)
		// Не возвращаем ошибку, фото уже отправлено
	}

	// Успех - формируем упрощённую metadata (event-sourcing)
	statusMetadata = buildImageSentMetadata(selectedImage.Filename, theme.String(), chatID, string(botID))

	s.Log.Info("image sent successfully",
		"theme", theme,
		"filename", selectedImage.Filename,
		"request_id", requestID,
		"chat_id", chatID)

	return nil
}

// buildImageSentMetadata создаёт упрощённую metadata для успешной отправки фото (event-sourcing)
func buildImageSentMetadata(filename, theme string, chatID int64, botID string) json.RawMessage {
	m := map[string]interface{}{
		"event": "image_sent",
		"image": map[string]interface{}{
			"filename": filename,
			"theme":    theme,
		},
		"telegram": map[string]interface{}{
			"chat_id": chatID,
		},
		"bot_id": botID,
	}

	data, err := json.Marshal(m)
	if err != nil {
		return nil
	}
	return json.RawMessage(data)
}
