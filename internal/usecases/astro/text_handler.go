package astro

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/admin/tg-bots/astro-bot/internal/usecases/astro/texts"
	"github.com/google/uuid"
)

// HandleText обрабатывает текстовые сообщения
func (s *Service) HandleText(ctx context.Context, botID domain.BotId, user *domain.User, text string, updateID int64) error {
	text = strings.TrimSpace(text)

	if text == "ПОДТВЕРДИТЬ" {
		return s.confirmResetBirthData(ctx, botID, user)
	}

	if s.isBirthDateInput(text) {
		return s.handleBirthDateInput(ctx, botID, user, text)
	}

	return s.handleUserQuestion(ctx, botID, user, text, updateID)
}

// isBirthDateInput проверяет, является ли текст полным вводом даты рождения
// Формат: ДД.ММ.ГГГГ чч:мм Город, КодСтраны или ДД.ММ.ГГГГ чч:мм Город
func (s *Service) isBirthDateInput(text string) bool {
	// Убираем code block
	text = strings.Trim(text, "`")
	text = strings.TrimSpace(text)

	// Разделяем по пробелам
	parts := strings.Fields(text)
	if len(parts) < 3 {
		return false
	}

	// Первая часть должна быть датой в формате ДД.ММ.ГГГГ
	datePart := parts[0]
	dateParts := strings.Split(datePart, ".")
	if len(dateParts) != 3 {
		return false
	}
	for _, part := range dateParts {
		if len(part) == 0 {
			return false
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return false
			}
		}
	}

	// Вторая часть должна быть временем в формате чч:мм
	timePart := parts[1]
	timeParts := strings.Split(timePart, ":")
	if len(timeParts) != 2 {
		return false
	}
	for _, part := range timeParts {
		if len(part) == 0 || len(part) > 2 {
			return false
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return false
			}
		}
	}

	// Третья и далее части - место рождения (должно быть хотя бы одно слово)
	if len(parts) < 3 {
		return false
	}

	return true
}

// handleBirthDateInput обрабатывает ввод даты рождения
// Формат: ДД.ММ.ГГГГ чч:мм Город, КодСтраны или ДД.ММ.ГГГГ чч:мм Город
func (s *Service) handleBirthDateInput(ctx context.Context, botID domain.BotId, user *domain.User, text string) error {
	if err := s.sendMessage(ctx, botID, user.TelegramChatID, texts.BirthDateCalculating); err != nil {
		s.Log.Warn("failed to send calculation message",
			"error", err,
			"user_id", user.ID,
		)
		// Продолжаем выполнение, даже если не удалось отправить сообщение
	}

	// Убираем обратные кавычки, если есть (code block)
	text = strings.Trim(text, "`")
	text = strings.TrimSpace(text)

	// Разделяем по пробелам
	parts := strings.Fields(text)
	if len(parts) < 3 {
		return s.sendMessage(ctx, botID, user.TelegramChatID, texts.BirthDateInvalidFormat)
	}

	birthDateTime, err := s.parseBirthDateTime(parts[0], parts[1])
	if err != nil {
		return s.sendMessage(ctx, botID, user.TelegramChatID, texts.BirthDateInvalidDateTime)
	}

	// Проверяем, что дата не в будущем
	if birthDateTime.After(time.Now()) {
		return s.sendMessage(ctx, botID, user.TelegramChatID, texts.BirthDateFuture)
	}

	// Парсим место рождения (объединяем все части после времени)
	birthPlace := strings.Join(parts[2:], " ")
	if birthPlace == "" {
		return s.sendMessage(ctx, botID, user.TelegramChatID, texts.BirthDateNoPlace)
	}

	// Сохраняем данные рождения
	now := time.Now()
	canChangeUntil := now.Add(24 * time.Hour)

	user.BirthDateTime = &birthDateTime
	birthPlaceStr := birthPlace
	user.BirthPlace = &birthPlaceStr
	user.BirthDataSetAt = &now
	user.BirthDataCanChangeUntil = &canChangeUntil
	user.UpdatedAt = now

	if err := s.UserRepo.Update(ctx, user); err != nil {
		s.Log.Error("failed to update birth data",
			"error", err,
			"user_id", user.ID,
		)
		return s.sendMessage(ctx, botID, user.TelegramChatID, texts.ErrorSaveData)
	}

	// Получаем натальную карту
	if err := s.fetchAndSaveNatalChart(ctx, user); err != nil {
		s.Log.Error("failed to fetch natal chart",
			"error", err,
			"user_id", user.ID,
		)
		return s.sendMessage(ctx, botID, user.TelegramChatID,
			texts.FormatBirthDateSuccessButChartError(
				birthDateTime.Format("02.01.2006"),
				birthDateTime.Format("15:04"),
				birthPlace,
			))
	}

	// Отправляем финальное сообщение об успехе
	return s.sendMessage(ctx, botID, user.TelegramChatID,
		texts.FormatBirthDateSuccess(
			birthDateTime.Format("02.01.2006"),
			birthDateTime.Format("15:04"),
			birthPlace,
		))
}

// parseBirthDateTime парсит дату и время из формата ДД.ММ.ГГГГ чч:мм
func (s *Service) parseBirthDateTime(dateStr, timeStr string) (time.Time, error) {
	// Парсим дату
	dateLayout := "02.01.2006"
	date, err := time.Parse(dateLayout, dateStr)
	if err != nil {
		return time.Time{}, err
	}

	// Парсим время
	timeLayout := "15:04"
	timePart, err := time.Parse(timeLayout, timeStr)
	if err != nil {
		return time.Time{}, err
	}

	// Объединяем дату и время
	birthDateTime := time.Date(
		date.Year(),
		date.Month(),
		date.Day(),
		timePart.Hour(),
		timePart.Minute(),
		0,
		0,
		time.UTC, // используем UTC, так как место рождения будет использовано для расчёта временной зоны
	)

	return birthDateTime, nil
}

// confirmResetBirthData подтверждает сброс даты рождения
func (s *Service) confirmResetBirthData(ctx context.Context, botID domain.BotId, user *domain.User) error {
	// Проверяем ещё раз, можно ли изменить
	if user.BirthDataCanChangeUntil == nil || time.Now().After(*user.BirthDataCanChangeUntil) {
		return s.sendMessage(ctx, botID, user.TelegramChatID, texts.ResetBirthDataLockedShort)
	}

	// Сбрасываем данные
	user.BirthDateTime = nil
	user.BirthPlace = nil
	user.BirthDataSetAt = nil
	user.BirthDataCanChangeUntil = nil
	user.NatalChartFetchedAt = nil
	user.UpdatedAt = time.Now()

	if err := s.UserRepo.Update(ctx, user); err != nil {
		s.Log.Error("failed to reset birth data",
			"error", err,
			"user_id", user.ID,
		)
		return s.sendMessage(ctx, botID, user.TelegramChatID, texts.ErrorResetData)
	}

	return s.sendMessageWithMarkdown(ctx, botID, user.TelegramChatID, texts.ResetBirthDataSuccess)
}

// handleUserQuestion обрабатывает вопрос пользователя
func (s *Service) handleUserQuestion(ctx context.Context, botID domain.BotId, user *domain.User, text string, updateID int64) (err error) {
	// Проверяем лимит бесплатных сообщений для бесплатных пользователей
	// Пользователь платный, если оплатил (is_paid) или получил доступ вручную (manual_granted)
	isPaidUser := user.IsPaid || user.ManualGranted
	if !isPaidUser && user.FreeMsgCount >= s.FreeMessagesLimit {
		if sendErr := s.sendMessage(ctx, botID, user.TelegramChatID, texts.PaymentLimitReached); sendErr != nil {
			s.Log.Warn("failed to send payment request message", "error", sendErr)
		}

		// Создаём платеж (invoice отправится автоматически)
		if s.PaymentService != nil {
			productID := "monthly_feed"
			productTitle := texts.BuyMonthlyFeedTitle
			description := texts.BuyMonthlyFeedDescription
			amount := s.StarsPrice

			_, paymentErr := s.PaymentService.CreatePayment(
				ctx,
				botID,
				user.ID,
				user.TelegramChatID,
				productID,
				productTitle,
				description,
				amount,
			)
			if paymentErr != nil {
				s.Log.Error("failed to create payment for free limit",
					"error", paymentErr,
					"user_id", user.ID,
					"bot_id", botID,
				)
				// Не возвращаем ошибку - сообщение уже отправлено
			}
		}

		return nil // Лимит достигнут, запрос не отправляем в RAG
	}

	var requestID uuid.UUID
	var statusStage domain.RequestStage
	var statusErrorCode string
	var statusMetadata json.RawMessage
	var statusCreated bool

	defer func() {
		if !statusCreated {
			return
		}

		if err != nil {
			// Ошибка - создаём статус ошибки
			errMsg := err.Error()
			if statusMetadata == nil {
				statusMetadata = domain.BuildErrorMetadata(
					statusStage,
					statusErrorCode,
					string(botID),
					nil,
				)
			}

			status := &domain.Status{
				ID:           uuid.New(),
				ObjectType:   domain.ObjectTypeRequest,
				ObjectID:     requestID,
				Status:       domain.StatusStatus(domain.RequestError),
				ErrorMessage: &errMsg,
				Metadata:     statusMetadata,
				CreatedAt:    time.Now(),
			}

			s.createOrLogStatus(ctx, status)
			s.sendAlertOrLog(ctx, status)
		} else {
			// успех отправки
			if statusMetadata == nil {
				return
			}

			status := &domain.Status{
				ID:         uuid.New(),
				ObjectType: domain.ObjectTypeRequest,
				ObjectID:   requestID,
				Status:     domain.StatusStatus(domain.RequestSentToRAG),
				Metadata:   statusMetadata,
				CreatedAt:  time.Now(),
			}
			s.createOrLogStatus(ctx, status)
		}
	}()

	// edge case - вопрос задаёт, а карты нет, если астроапи отдало ошибку, но дата сохранена - догружаем по ходу
	if user.NatalChartFetchedAt == nil {
		if err = s.fetchAndSaveNatalChart(ctx, user); err != nil {
			statusStage = domain.StageLoadNatalChart
			statusErrorCode = "NATAL_CHART_NOT_FOUND"
			s.Log.Error("failed to fetch natal chart",
				"error", err,
				"user_id", user.ID,
			)
			originalErr := err
			if sendErr := s.sendMessage(ctx, botID, user.TelegramChatID, texts.ErrorNatalChartNotFound); sendErr != nil {
				s.Log.Warn("failed to notify user about error", "error", sendErr)
			}
			return originalErr
		}
	}

	request := &domain.Request{
		ID:          uuid.New(),
		UserID:      user.ID,
		BotID:       botID,
		TGUpdateID:  &updateID,
		RequestType: domain.RequestTypeUser, // обычный запрос от пользователя
		RequestText: text,
		CreatedAt:   time.Now(),
	}

	if err = s.RequestRepo.Create(ctx, request); err != nil {
		requestID = request.ID
		statusCreated = true
		statusStage = domain.StageCreateRequest
		statusErrorCode = "DB_CREATE_ERROR"
		s.Log.Error("failed to create request",
			"error", err,
			"user_id", user.ID,
			"update_id", updateID,
		)
		originalErr := err
		if sendErr := s.sendMessage(ctx, botID, user.TelegramChatID, texts.ErrorCreateRequest); sendErr != nil {
			s.Log.Warn("failed to notify user about error", "error", sendErr)
		}
		return originalErr
	}

	requestID = request.ID
	statusCreated = true

	// Инкрементируем счётчик бесплатных сообщений для бесплатных пользователей
	if !isPaidUser {
		if err = s.UserRepo.UpdateFreeMsgCount(ctx, user.ID); err != nil {
			s.Log.Warn("failed to increment free_msg_count",
				"error", err,
				"user_id", user.ID,
				"request_id", requestID,
			)
			// Не возвращаем ошибку - продолжаем обработку запроса
		}
	}

	// lazy loading - отчёт достаём ток перед отправкой в кафку
	natalReport, err := s.UserRepo.GetNatalChart(ctx, user.ID)
	if err != nil {
		statusStage = domain.StageLoadNatalChart
		statusErrorCode = "NATAL_CHART_NOT_FOUND"
		s.Log.Error("failed to get natal report for RAG",
			"error", err,
			"user_id", user.ID,
			"request_id", requestID,
		)
		originalErr := err
		if sendErr := s.sendMessage(ctx, botID, user.TelegramChatID, texts.ErrorGetNatalReport); sendErr != nil {
			s.Log.Warn("failed to notify user about error", "error", sendErr)
		}
		return originalErr
	}

	if s.KafkaProducer != nil {
		partition, offset, err := s.KafkaProducer.SendRAGRequest(ctx, request.ID, request.BotID, user.TelegramChatID, request.RequestText, natalReport)
		if err != nil {
			statusStage = domain.StageKafkaSend
			statusErrorCode = "KAFKA_SEND_ERROR"
			if strings.Contains(err.Error(), "timeout") {
				statusErrorCode = "KAFKA_TIMEOUT"
			} else if strings.Contains(err.Error(), "connection") {
				statusErrorCode = "KAFKA_CONN_ERROR"
			}
			s.Log.Error("failed to send request to kafka",
				"error", err,
				"request_id", requestID,
				"user_id", user.ID,
			)
			originalErr := err
			if sendErr := s.sendMessage(ctx, botID, user.TelegramChatID, texts.ErrorSendRequest); sendErr != nil {
				s.Log.Warn("failed to notify user about error", "error", sendErr)
			}
			return originalErr
		}

		// успех отправки
		statusMetadata = domain.BuildKafkaMetadata(
			"requests",
			partition,
			offset,
			string(botID),
			len(text),
			len(natalReport),
		)

		s.Log.Info("request sent to kafka",
			"request_id", requestID,
			"partition", partition,
			"offset", offset,
		)
	} else {
		s.Log.Warn("kafka producer not configured, skipping RAG request",
			"request_id", requestID,
		)
	}

	return s.sendMessage(ctx, botID, user.TelegramChatID, texts.UserQuestionReceived)
}
