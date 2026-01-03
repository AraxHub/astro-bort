package astro

import (
	"context"
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

// isBirthDateInput проверяет, является ли текст датой в формате ДД.ММ.ГГГГ
func (s *Service) isBirthDateInput(text string) bool {
	// Простая проверка формата (можно улучшить)
	parts := strings.Split(text, ".")
	if len(parts) != 3 {
		return false
	}
	// Проверяем, что все части - числа
	for _, part := range parts {
		if len(part) == 0 {
			return false
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}

// handleBirthDateInput обрабатывает ввод даты рождения
func (s *Service) handleBirthDateInput(ctx context.Context, botID domain.BotId, user *domain.User, text string) error {
	// Парсим дату
	birthDate, err := s.parseBirthDate(text)
	if err != nil {
		return s.sendMessage(ctx, botID, user.TelegramChatID,
			"❌ Неверный формат даты\n"+
				"Введи дату в формате ДД.ММ.ГГГГ")
	}

	// Проверяем, что дата не в будущем
	if birthDate.After(time.Now()) {
		return s.sendMessage(ctx, botID, user.TelegramChatID,
			"❌ Дата рождения не может быть в будущем")
	}

	// Сохраняем дату рождения
	now := time.Now()
	canChangeUntil := now.Add(24 * time.Hour)

	user.BirthDateTime = &birthDate
	user.BirthDataSetAt = &now
	user.BirthDataCanChangeUntil = &canChangeUntil
	user.UpdatedAt = now

	if err := s.UserRepo.Update(ctx, user); err != nil {
		s.Log.Error("failed to update birth date",
			"error", err,
			"user_id", user.ID,
		)
		return s.sendMessage(ctx, botID, user.TelegramChatID, "❌ Ошибка при сохранении даты")
	}

	// Пытаемся получить натальную карту
	if err := s.fetchAndSaveNatalChart(ctx, user); err != nil {
		s.Log.Error("failed to fetch natal chart",
			"error", err,
			"user_id", user.ID,
		)
		return s.sendMessage(ctx, botID, user.TelegramChatID,
			"✅ Дата установлена\n"+
				"⚠️ Можно изменить в течение 24ч\n\n"+
				"❌ Не удалось получить натальную карту. Попробуй позже.")
	}

	return s.sendMessage(ctx, botID, user.TelegramChatID,
		"✅ Дата установлена\n"+
			"⚠️ Можно изменить в течение 24ч\n\n"+
			"✅ Натальная карта получена!\nГотов к работе")
}

// parseBirthDate парсит дату из формата ДД.ММ.ГГГГ
func (s *Service) parseBirthDate(text string) (time.Time, error) {
	layout := "02.01.2006"
	return time.Parse(layout, text)
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

	return s.sendMessage(ctx, botID, user.TelegramChatID,
		"✅ Дата рождения и натальная карта сброшены\n\n"+
			"Введи новую дату в формате ДД.ММ.ГГГГ")
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
