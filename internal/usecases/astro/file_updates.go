package astro

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/google/uuid"
)

// SyncImagesResult результат синхронизации картинок
type SyncImagesResult struct {
	Processed int
	Created   int
	Errors    []string
}

// SyncImages синхронизирует картинки из S3 в Telegram и БД
// themes - список тем для синхронизации (если пустой, синхронизируются все темы)
func (s *Service) SyncImages(ctx context.Context, botID domain.BotId, syncChatID int64, messageThreadID *int64, themes []string) (*SyncImagesResult, error) {
	if s.S3Client == nil {
		return nil, fmt.Errorf("S3 client not configured")
	}

	if s.ImageRepo == nil {
		return nil, fmt.Errorf("image repository not configured")
	}

	result := &SyncImagesResult{
		Errors: []string{},
	}

	// Определяем список тем для синхронизации
	var themesToSync []domain.ImageTheme
	if len(themes) == 0 {
		// Если темы не указаны - синхронизируем все
		themesToSync = domain.AllImageThemes()
	} else {
		// Конвертируем строки в ImageTheme и валидируем
		for _, themeStr := range themes {
			theme := domain.ImageTheme(themeStr)
			if !theme.IsValid() {
				result.Errors = append(result.Errors, fmt.Sprintf("Invalid theme: %s", themeStr))
				continue
			}
			themesToSync = append(themesToSync, theme)
		}
	}

	if len(themesToSync) == 0 {
		return nil, fmt.Errorf("no valid themes to sync")
	}

	// Получаем все файлы из S3 (сканируем все темы)
	allFiles := make(map[string]string) // filename -> theme

	for _, theme := range themesToSync {
		prefix := theme.S3Path()
		files, err := s.S3Client.ListFiles(ctx, prefix)
		if err != nil {
			s.Log.Error("failed to list files from S3",
				"error", err,
				"theme", theme,
				"prefix", prefix)
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to list files for theme %s: %v", theme, err))
			continue
		}

		for _, filePath := range files {
			// Извлекаем имя файла из пути: themes/Love/L1.jpg -> L1.jpg
			filename := filepath.Base(filePath)
			allFiles[filename] = theme.String()
		}
	}

	s.Log.Info("found files in S3",
		"total_files", len(allFiles),
		"themes", themesToSync)

	// Получаем все существующие картинки из БД
	existingImages, err := s.ImageRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing images: %w", err)
	}

	existingFilenames := make(map[string]bool)
	for _, img := range existingImages {
		existingFilenames[img.Filename] = true
	}

	// Обрабатываем новые файлы
	firstPhoto := true
	for filename, theme := range allFiles {
		result.Processed++

		// Пропускаем если уже есть в БД
		if existingFilenames[filename] {
			s.Log.Debug("image already exists in DB", "filename", filename)
			continue
		}

		// Задержка перед отправкой (кроме первой фотки) для соблюдения rate limit Telegram
		if !firstPhoto {
			select {
			case <-ctx.Done():
				return result, ctx.Err()
			case <-time.After(1100 * time.Millisecond): // 1.1 секунды между отправками
			}
		}
		firstPhoto = false

		// Получаем файл из S3
		themeObj := domain.ImageTheme(theme)
		filePath := themeObj.S3Path() + filename
		fileData, err := s.S3Client.GetFile(ctx, filePath)
		if err != nil {
			s.Log.Error("failed to get file from S3",
				"error", err,
				"file_path", filePath)
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to get %s: %v", filename, err))
			continue
		}

		// Отправляем в Telegram и получаем file_id
		tgFileID, err := s.sendPhotoToTelegram(ctx, botID, syncChatID, messageThreadID, fileData, filename)
		if err != nil {
			s.Log.Error("failed to send photo to Telegram",
				"error", err,
				"filename", filename)
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to send %s to Telegram: %v", filename, err))
			continue
		}

		// Сохраняем в БД
		image := &domain.Image{
			ID:        uuid.New(),
			Filename:  filename,
			TgFileID:  tgFileID,
			Theme:     &themeObj,
			CreatedAt: time.Now(),
		}

		if err := s.ImageRepo.Create(ctx, image); err != nil {
			s.Log.Error("failed to save image to DB",
				"error", err,
				"filename", filename)
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to save %s to DB: %v", filename, err))
			continue
		}

		result.Created++
		s.Log.Info("image synced successfully",
			"filename", filename,
			"theme", theme,
			"tg_file_id", tgFileID)
	}

	return result, nil
}

// sendPhotoToTelegram отправляет фото в Telegram и возвращает file_id
func (s *Service) sendPhotoToTelegram(ctx context.Context, botID domain.BotId, chatID int64, messageThreadID *int64, photoData []byte, filename string) (string, error) {
	if s.TelegramService == nil {
		return "", fmt.Errorf("telegram service not configured")
	}

	fileID, err := s.TelegramService.SendPhoto(ctx, botID, chatID, messageThreadID, photoData, filename)
	if err != nil {
		return "", fmt.Errorf("failed to send photo: %w", err)
	}

	return fileID, nil
}
