package astro

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
)

func (s *Service) HandleCommand(ctx context.Context, botID domain.BotId, user *domain.User, command string, updateID int64) error {
	switch command {
	case "start":
		return s.HandleStart(ctx, botID, user, updateID)
	case "help":
		return s.HandleHelp(ctx, botID, user)
	case "my_info":
		return s.HandleMyInfo(ctx, botID, user)
	case "reset_birth_data":
		return s.HandleResetBirthData(ctx, botID, user)
	default:
		return s.sendMessage(ctx, botID, user.TelegramChatID, fmt.Sprintf("❌ Неизвестная команда: /%s\n\nИспользуй /help для списка команд", command))
	}
}

func (s *Service) HandleStart(ctx context.Context, botID domain.BotId, user *domain.User, updateID int64) error {

	if user.BirthDateTime == nil {
		message := "⚠️ Дата рождения устанавливается ОДИН РАЗ\n" +
			"Задавай вопросы только от своего лица\n\n" +
			"Введи дату рождения в формате ДД.ММ.ГГГГ чч:мм:00 Город"
		return s.sendMessage(ctx, botID, user.TelegramChatID, message)
	}

	// Проверяем наличие натальной карты (ленивая загрузка - проверяем флаг, не загружаем данные)
	if user.NatalChartFetchedAt == nil {
		// Пытаемся получить натальную карту
		if err := s.fetchAndSaveNatalChart(ctx, user); err != nil {
			s.Log.Error("failed to fetch natal chart",
				"error", err,
				"user_id", user.ID,
			)
			return s.sendMessage(ctx, botID, user.TelegramChatID, "❌ Не удалось получить натальную карту. Попробуйте позже.")
		}
	}

	message := "✅ Натальная карта получена!\nГотов к работе\n\n" +
		"Задавай вопросы)"
	return s.sendMessage(ctx, botID, user.TelegramChatID, message)
}

// HandleHelp обрабатывает команду /help
func (s *Service) HandleHelp(ctx context.Context, botID domain.BotId, user *domain.User) error {
	message := "📋 Доступные команды:\n\n" +
		"/start - Начать работу\n" +
		"/reset_birth_data - Сбросить дату рождения (только в течение 24 часов)\n" +
		"/my_info - Моя информация\n" +
		"/help - Показать эту справку"
	return s.sendMessage(ctx, botID, user.TelegramChatID, message)
}

// HandleMyInfo обрабатывает команду /my_info
func (s *Service) HandleMyInfo(ctx context.Context, botID domain.BotId, user *domain.User) error {
	var message strings.Builder
	message.WriteString("ℹ️ Твоя информация:\n\n")

	if user.BirthDateTime != nil {
		message.WriteString(fmt.Sprintf("📅 Дата рождения: %s\n", user.BirthDateTime.Format("02.01.2006 15:04")))
		if user.BirthPlace != nil {
			message.WriteString(fmt.Sprintf("📍 Место рождения: %s\n", *user.BirthPlace))
		}
	} else {
		message.WriteString("📅 Дата рождения: не установлена\n")
	}

	if user.NatalChartFetchedAt != nil {
		message.WriteString("✨ Натальная карта: ✅\n")
		message.WriteString(fmt.Sprintf("   Получена: %s\n", user.NatalChartFetchedAt.Format("02.01.2006 15:04")))
	} else {
		message.WriteString("✨ Натальная карта: ❌\n")
	}

	return s.sendMessage(ctx, botID, user.TelegramChatID, message.String())
}

// HandleResetBirthData обрабатывает команду /reset_birth_data
func (s *Service) HandleResetBirthData(ctx context.Context, botID domain.BotId, user *domain.User) error {
	// Проверяем, можно ли изменить дату (в течение 24 часов)
	if user.BirthDataCanChangeUntil == nil || time.Now().After(*user.BirthDataCanChangeUntil) {
		return s.sendMessage(ctx, botID, user.TelegramChatID,
			"❌ Дата заблокирована\n"+
				"Обратись к администратору для изменения")
	}

	// Запрашиваем подтверждение
	message := "⚠️ Ты уверен?\n\n" +
		"Это удалит дату рождения и натальную карту.\n" +
		"Введи 'ПОДТВЕРДИТЬ' для подтверждения."
	return s.sendMessage(ctx, botID, user.TelegramChatID, message)
}
