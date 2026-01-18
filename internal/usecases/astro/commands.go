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
		return s.HandleStart(ctx, botID, user)
	case "help":
		return s.HandleHelp(ctx, botID, user)
	case "my_info":
		return s.HandleMyInfo(ctx, botID, user)
	case "reset_birth_data":
		return s.HandleResetBirthData(ctx, botID, user)
	case "buy", "test_payment":
		return s.HandleBuy(ctx, botID, user)
	default:
		return s.sendMessage(ctx, botID, user.TelegramChatID, fmt.Sprintf("❌ Неизвестная команда: /%s\n\nИспользуй /help для списка команд", command))
	}
}

func (s *Service) HandleStart(ctx context.Context, botID domain.BotId, user *domain.User) error {
	if user.BirthDateTime == nil {
		message := "🐱 Привет! Я Кита, твоя астрологиня ✨\n\n" +
			"Я помогу тебе разобраться в твоей натальной карте и ответить на вопросы о жизни, отношениях, карьере и многом другом.\n\n" +
			"⚠️ Важно:\n" +
			"• Я работаю только с твоей натальной картой\n" +
			"• Задавай вопросы только от своего лица\n" +
			"• Если начнёшь спрашивать от лица других людей, я запутаюсь в картах и не смогу дать точные ответы\n\n" +
			"📅 Для расчёта натальной карты мне нужны твои данные рождения.\n\n" +
			"Я подготовила для тебя формочку - скопируй её и замени значения на свои, отправить можешь обычным текстом:\n\n" +
			"Нажми на неё и она скопируется:\n" +
			"```\n15.03.1990 14:30 Москва, RU\n```\n\n" +
			"💡 Если не знаешь код страны, просто укажи город:\n" +
			"```\n15.03.1990 14:30 Москва\n```\n\n" +
			"⚠️ Дата устанавливается ОДИН РАЗ. Если ошибёшься, можешь изменить её в течение 24 часов через команду /reset\\_birth\\_data"
		return s.sendMessageWithMarkdown(ctx, botID, user.TelegramChatID, message)
	}

	// edge case - дата есть, карты нет, пытаемся рассчитать
	if user.NatalChartFetchedAt == nil {
		if err := s.fetchAndSaveNatalChart(ctx, user); err != nil {
			s.Log.Error("failed to fetch natal chart",
				"error", err,
				"user_id", user.ID,
			)
			return s.sendMessage(ctx, botID, user.TelegramChatID, "❌ Не удалось рассчитать натальную карту. Попробуйте позже.")
		}
	}

	message := "🐱 Привет снова! Я Кита, твоя астрологиня ✨\n\n" +
		"Твоя натальная карта уже рассчитана, я готова отвечать на вопросы!\n\n" +
		"⚠️ Напоминаю:\n" +
		"• Я работаю только с твоей натальной картой\n" +
		"• Задавай вопросы только от своего лица\n" +
		"• Если начнёшь спрашивать про других людей, я запутаюсь в картах\n\n" +
		"💡 Твоя дата рождения установлена один раз. Если ошибся, можешь изменить её в течение 24 часов через команду /reset_birth_data\n\n" +
		"Готов ответить на твои вопросы! 🚀"
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

	// Проверяем реальное наличие карты в БД, а не только флаг
	natalReport, err := s.UserRepo.GetNatalChart(ctx, user.ID)
	if err != nil {
		s.Log.Error("failed to get natal chart for my_info",
			"error", err,
			"user_id", user.ID,
		)
		message.WriteString("✨ Натальная карта: ❌ (ошибка при проверке)\n")
	} else if len(natalReport) > 0 {
		message.WriteString("✨ Натальная карта: ✅\n")
		if user.NatalChartFetchedAt != nil {
			message.WriteString(fmt.Sprintf("   Получена: %s\n", user.NatalChartFetchedAt.Format("02.01.2006 15:04")))
		}
	} else {
		message.WriteString("✨ Натальная карта: ❌ (не установлена)\n")
		if user.BirthDateTime != nil && user.BirthPlace != nil {
			message.WriteString("   Используй /start для расчёта карты\n")
		} else {
			message.WriteString("   Используй /reset_birth_data для настройки\n")
		}
	}

	message.WriteString("\n")

	// Информация о тарифе и бесплатных сообщениях
	isPaidUser := user.IsPaid || user.ManualGranted
	if isPaidUser {
		message.WriteString("💎 Тариф: куплен 🐾\n")
		if !user.ManualGranted && s.PaymentRepo != nil {
			// Получаем дату последнего успешного платежа для вычисления даты окончания
			lastPaymentDate, err := s.PaymentRepo.GetLastSuccessfulPaymentDate(ctx, user.ID)
			if err != nil {
				s.Log.Warn("failed to get last payment date for my_info",
					"error", err,
					"user_id", user.ID,
				)
				message.WriteString("   Тариф активен 🎉\n")
			} else if lastPaymentDate != nil {
				// Вычисляем дату окончания: последний платёж + 30 дней
				expiryDate := lastPaymentDate.Add(30 * 24 * time.Hour)
				message.WriteString("🆓 Бесплатных сообщений осталось: безлимит 🐱\n")
				message.WriteString(fmt.Sprintf("   Тариф активен до %s 🎉\n", expiryDate.Format("02.01.2006")))
			} else {
				message.WriteString("   Тариф активен 🎉\n")
			}
		} else if user.ManualGranted {
			message.WriteString("🆓 Бесплатных сообщений осталось: безлимит 🐱\n")
			message.WriteString("   Тариф активен (ручной доступ) 🎉\n")
		} else {
			message.WriteString("🆓 Бесплатных сообщений осталось: безлимит 🐱\n")
			message.WriteString("   Тариф активен 🎉\n")
		}
	} else {
		message.WriteString("💎 Тариф: не куплен 🐾\n")
		remaining := s.FreeMessagesLimit - user.FreeMsgCount
		if remaining < 0 {
			remaining = 0
		}
		message.WriteString(fmt.Sprintf("🆓 Бесплатных сообщений осталось: %d из %d 🐱\n", remaining, s.FreeMessagesLimit))
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

	message := "⚠️ Ты уверен?\n\n" +
		"Это удалит дату рождения и натальную карту.\n" +
		"Введи 'ПОДТВЕРДИТЬ' для подтверждения."
	return s.sendMessage(ctx, botID, user.TelegramChatID, message)
}

// HandleBuy обрабатывает команду /buy или /test_payment (тестовый платёж)
func (s *Service) HandleBuy(ctx context.Context, botID domain.BotId, user *domain.User) error {
	if s.PaymentService == nil {
		return s.sendMessage(ctx, botID, user.TelegramChatID,
			"❌ Платёжная система недоступна")
	}

	// Тестовые данные платежа
	productID := "test_premium"
	productTitle := "Премиум доступ (тест)"
	description := "Тестовый платёж для проверки системы Stars. Доступ на 1 месяц."
	amount := s.StarsPrice // цена из конфигурации

	payment, err := s.PaymentService.CreatePayment(
		ctx,
		botID,
		user.ID,
		user.TelegramChatID,
		productID,
		productTitle,
		description,
		amount,
	)
	if err != nil {
		s.Log.Error("failed to create payment",
			"error", err,
			"user_id", user.ID,
			"bot_id", botID,
		)
		return s.sendMessage(ctx, botID, user.TelegramChatID,
			"❌ Не удалось создать платёж. Попробуйте позже.")
	}

	s.Log.Info("test payment created",
		"payment_id", payment.ID,
		"user_id", user.ID,
		"amount", amount,
	)
	return nil
}
