package astro

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/google/uuid"
)

// HandleText обрабатывает текстовые сообщения
func (s *Service) HandleText(ctx context.Context, botID domain.BotId, user *domain.User, text string, updateID int64) error {
	text = strings.TrimSpace(text)

	// Проверяем, является ли это подтверждением сброса даты
	if text == "ПОДТВЕРДИТЬ" {
		return s.confirmResetBirthData(ctx, botID, user)
	}

	// Проверяем, является ли это датой рождения (формат ДД.ММ.ГГГГ)
	if s.isBirthDateInput(text) {
		return s.handleBirthDateInput(ctx, botID, user, text)
	}

	// Обычное текстовое сообщение - создаём запрос
	return s.handleUserQuestion(ctx, botID, user, text, updateID)
}

// isBirthDateInput проверяет, является ли текст полным вводом даты рождения
// Формат: ДД.ММ.ГГГГ чч:мм Город, КодСтраны или ДД.ММ.ГГГГ чч:мм Город
func (s *Service) isBirthDateInput(text string) bool {
	// Убираем обратные кавычки, если есть (code block)
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
	// Сразу отправляем ответ, что начинаем расчёт
	if err := s.sendMessage(ctx, botID, user.TelegramChatID, "✨ Рассчитываю твою натальную карту..."); err != nil {
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
		return s.sendMessage(ctx, botID, user.TelegramChatID,
			"❌ Неверный формат\n\n"+
				"Используй формат:\n"+
				"`ДД.ММ.ГГГГ чч:мм Город, КодСтраны`\n\n"+
				"Пример:\n"+
				"`15.03.1990 14:30 Москва, RU`")
	}

	// Парсим дату и время
	birthDateTime, err := s.parseBirthDateTime(parts[0], parts[1])
	if err != nil {
		return s.sendMessage(ctx, botID, user.TelegramChatID,
			"❌ Неверный формат даты или времени\n\n"+
				"Используй формат:\n"+
				"`ДД.ММ.ГГГГ чч:мм Город, КодСтраны`\n\n"+
				"Пример:\n"+
				"`15.03.1990 14:30 Москва, RU`")
	}

	// Проверяем, что дата не в будущем
	if birthDateTime.After(time.Now()) {
		return s.sendMessage(ctx, botID, user.TelegramChatID,
			"❌ Дата рождения не может быть в будущем")
	}

	// Парсим место рождения (объединяем все части после времени)
	birthPlace := strings.Join(parts[2:], " ")
	if birthPlace == "" {
		return s.sendMessage(ctx, botID, user.TelegramChatID,
			"❌ Не указано место рождения\n\n"+
				"Используй формат:\n"+
				"`ДД.ММ.ГГГГ чч:мм Город, КодСтраны`")
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
		return s.sendMessage(ctx, botID, user.TelegramChatID, "❌ Ошибка при сохранении данных")
	}

	// Получаем натальную карту
	if err := s.fetchAndSaveNatalChart(ctx, user); err != nil {
		s.Log.Error("failed to fetch natal chart",
			"error", err,
			"user_id", user.ID,
		)
		return s.sendMessage(ctx, botID, user.TelegramChatID,
			"✅ Данные приняты:\n"+
				fmt.Sprintf("📅 Дата: %s\n", birthDateTime.Format("02.01.2006"))+
				fmt.Sprintf("🕐 Время: %s\n", birthDateTime.Format("15:04"))+
				fmt.Sprintf("📍 Место: %s\n\n", birthPlace)+
				"⚠️ Можно изменить в течение 24ч\n\n"+
				"❌ Не удалось рассчитать натальную карту. Попробуй позже через /reset_birth_data.")
	}

	// Отправляем финальное сообщение об успехе
	return s.sendMessage(ctx, botID, user.TelegramChatID,
		"🎉 Готово! Твоя натальная карта рассчитана!\n\n"+
			"✅ Данные:\n"+
			fmt.Sprintf("📅 Дата: %s\n", birthDateTime.Format("02.01.2006"))+
			fmt.Sprintf("🕐 Время: %s\n", birthDateTime.Format("15:04"))+
			fmt.Sprintf("📍 Место: %s\n\n", birthPlace)+
			"⚠️ Можно изменить в течение 24ч\n\n"+
			"Теперь можешь задавать вопросы, и я буду отвечать на основе твоей карты.\n\n"+
			"💡 Помни: я работаю только с твоей картой, поэтому задавай вопросы от своего лица.\n\n"+
			"Начни с любого вопроса! 🚀")
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
		0,        // секунды = 0
		0,        // наносекунды = 0
		time.UTC, // используем UTC, так как место рождения будет использовано для расчёта временной зоны
	)

	return birthDateTime, nil
}

// confirmResetBirthData подтверждает сброс даты рождения
func (s *Service) confirmResetBirthData(ctx context.Context, botID domain.BotId, user *domain.User) error {
	// Проверяем ещё раз, можно ли изменить
	if user.BirthDataCanChangeUntil == nil || time.Now().After(*user.BirthDataCanChangeUntil) {
		return s.sendMessage(ctx, botID, user.TelegramChatID,
			"❌ Дата заблокирована\n"+
				"Обратись к администратору")
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
		return s.sendMessage(ctx, botID, user.TelegramChatID, "❌ Ошибка при сбросе данных")
	}

	message := "✅ Дата рождения и натальная карта сброшены\n\n" +
		"Введи новые данные в формате:\n\n" +
		"`ДД.ММ.ГГГГ чч:мм Город, КодСтраны`\n\n" +
		"Пример:\n" +
		"```\n15.03.1990 14:30 Москва, RU\n```"
	return s.sendMessageWithMarkdown(ctx, botID, user.TelegramChatID, message)
}

// handleUserQuestion обрабатывает вопрос пользователя
// todo рефактор - отправка в раг
func (s *Service) handleUserQuestion(ctx context.Context, botID domain.BotId, user *domain.User, text string, updateID int64) error {
	// Проверяем наличие натальной карты (ленивая загрузка - проверяем флаг, не загружаем данные)
	if user.NatalChartFetchedAt == nil {
		// Пытаемся получить натальную карту
		if err := s.fetchAndSaveNatalChart(ctx, user); err != nil {
			s.Log.Error("failed to fetch natal chart",
				"error", err,
				"user_id", user.ID,
			)
			return s.sendMessage(ctx, botID, user.TelegramChatID,
				"❌ Натальная карта не найдена\n"+
					"Используй /start для настройки")
		}
	}

	// Создаём запрос
	request := &domain.Request{
		ID:          uuid.New(),
		UserID:      user.ID,
		BotID:       botID,
		TGUpdateID:  &updateID,
		RequestText: text,
		CreatedAt:   time.Now(),
	}

	if err := s.RequestRepo.Create(ctx, request); err != nil {
		s.Log.Error("failed to create request",
			"error", err,
			"user_id", user.ID,
			"update_id", updateID,
		)
		return s.sendMessage(ctx, botID, user.TelegramChatID, "❌ Ошибка при создании запроса")
	}

	// Ленивая загрузка: загружаем natal_chart только когда нужно отправить в RAG
	natalChart, err := s.UserRepo.GetNatalChart(ctx, user.ID)
	if err != nil {
		s.Log.Error("failed to get natal chart for RAG",
			"error", err,
			"user_id", user.ID,
			"request_id", request.ID,
		)
		// Продолжаем без natal_chart или возвращаем ошибку - зависит от требований
		// Пока логируем и продолжаем
	}

	// TODO: отправить в Kafka для RAG (с natal_chart)
	s.Log.Info("request created",
		"request_id", request.ID,
		"user_id", user.ID,
		"text_length", len(text),
		"natal_chart_size", len(natalChart),
	)

	return s.sendMessage(ctx, botID, user.TelegramChatID,
		"✅ Запрос получен\nОбрабатываю...")
}
