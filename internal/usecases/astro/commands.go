package astro

import (
	"context"
	"time"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/admin/tg-bots/astro-bot/internal/usecases/astro/texts"
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
		return s.sendMessage(ctx, botID, user.TelegramChatID, texts.FormatUnknownCommand(command))
	}
}

func (s *Service) HandleStart(ctx context.Context, botID domain.BotId, user *domain.User) error {
	if user.BirthDateTime == nil {
		return s.sendMessageWithMarkdown(ctx, botID, user.TelegramChatID, texts.StartFirstTime)
	}

	// edge case - дата есть, карты нет, пытаемся рассчитать
	if user.NatalChartFetchedAt == nil {
		if err := s.fetchAndSaveNatalChart(ctx, user); err != nil {
			s.Log.Error("failed to fetch natal chart",
				"error", err,
				"user_id", user.ID,
			)
			return s.sendMessage(ctx, botID, user.TelegramChatID, texts.ErrorNatalChartCalculation)
		}
	}

	return s.sendMessage(ctx, botID, user.TelegramChatID, texts.StartReturning)
}

// HandleHelp обрабатывает команду /help
func (s *Service) HandleHelp(ctx context.Context, botID domain.BotId, user *domain.User) error {
	return s.sendMessage(ctx, botID, user.TelegramChatID, texts.HelpCommand)
}

// HandleMyInfo обрабатывает команду /my_info
func (s *Service) HandleMyInfo(ctx context.Context, botID domain.BotId, user *domain.User) error {
	// Проверяем реальное наличие карты в БД, а не только флаг
	natalReport, err := s.UserRepo.GetNatalChart(ctx, user.ID)
	var natalChartExists bool
	var natalChartFetchedAt *time.Time
	if err != nil {
		s.Log.Error("failed to get natal chart for my_info",
			"error", err,
			"user_id", user.ID,
		)
		// В случае ошибки считаем, что карты нет, но показываем ошибку
		natalChartExists = false
	} else {
		natalChartExists = len(natalReport) > 0
		if natalChartExists {
			natalChartFetchedAt = user.NatalChartFetchedAt
		}
	}

	var expiryDate *time.Time
	isPaidUser := user.IsPaid || user.ManualGranted
	if isPaidUser && !user.ManualGranted && s.PaymentRepo != nil {
		lastPaymentDate, err := s.PaymentRepo.GetLastSuccessfulPaymentDate(ctx, user.ID)
		if err != nil {
			s.Log.Warn("failed to get last payment date for my_info",
				"error", err,
				"user_id", user.ID,
			)
		} else if lastPaymentDate != nil {
			exp := lastPaymentDate.Add(30 * 24 * time.Hour)
			expiryDate = &exp
		}
	}

	message := texts.FormatMyInfo(
		user.BirthDateTime,
		user.BirthPlace,
		natalChartExists,
		natalChartFetchedAt,
		isPaidUser,
		user.ManualGranted,
		user.FreeMsgCount,
		s.FreeMessagesLimit,
		expiryDate,
	)

	return s.sendMessage(ctx, botID, user.TelegramChatID, message)
}

// HandleResetBirthData обрабатывает команду /reset_birth_data
func (s *Service) HandleResetBirthData(ctx context.Context, botID domain.BotId, user *domain.User) error {
	// Проверяем, можно ли изменить дату (в течение 24 часов)
	if user.BirthDataCanChangeUntil == nil || time.Now().After(*user.BirthDataCanChangeUntil) {
		return s.sendMessage(ctx, botID, user.TelegramChatID, texts.ResetBirthDataLocked)
	}

	return s.sendMessage(ctx, botID, user.TelegramChatID, texts.ResetBirthDataConfirm)
}

// HandleBuy обрабатывает команду /buy или /test_payment (тестовый платёж)
func (s *Service) HandleBuy(ctx context.Context, botID domain.BotId, user *domain.User) error {
	if s.PaymentService == nil {
		return s.sendMessage(ctx, botID, user.TelegramChatID, texts.PaymentServiceUnavailable)
	}

	// Тестовые данные платежа
	productID := "test_premium"
	productTitle := texts.BuyTestProductTitle
	description := texts.BuyTestProductDescription
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
		return s.sendMessage(ctx, botID, user.TelegramChatID, texts.PaymentCreateError)
	}

	s.Log.Info("test payment created",
		"payment_id", payment.ID,
		"user_id", user.ID,
		"amount", amount,
	)
	return nil
}
