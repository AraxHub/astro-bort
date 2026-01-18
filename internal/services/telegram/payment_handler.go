package telegram

import (
	"context"
	"fmt"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/google/uuid"
)

// HandlePreCheckoutQuery обрабатывает pre_checkout_query от Telegram (для платежей Stars)
func (s *Service) HandlePreCheckoutQuery(ctx context.Context, botID domain.BotId, query *domain.PreCheckoutQuery) error {
	if query == nil || query.From == nil {
		s.Log.Error("pre_checkout_query is nil or has no from")
		return fmt.Errorf("invalid pre_checkout_query")
	}

	botType, err := s.GetBotType(botID)
	if err != nil {
		return fmt.Errorf("failed to get bot_type for bot_id %s: %w", botID, err)
	}

	// Получаем use case для бота
	botService, ok := s.BotTypeToUsecase[botType]
	if !ok {
		return fmt.Errorf("unknown bot_type: %s", botType)
	}

	// Получаем или создаём пользователя
	user, err := botService.GetOrCreateUser(ctx, botID, query.From, nil) // Chat не нужен для pre_checkout_query
	if err != nil {
		return domain.WrapBusinessError(fmt.Errorf("failed to get or create user: %w", err))
	}

	if s.PaymentUseCase == nil {
		s.Log.Warn("payment use case not configured, rejecting pre_checkout_query",
			"query_id", query.ID,
		)
		return fmt.Errorf("payment use case not configured")
	}

	// Вызываем payment use case для обработки pre_checkout_query
	confirmed, err := s.PaymentUseCase.HandlePreCheckoutQuery(
		ctx,
		botID,
		query.ID,
		user.ID,
		query.TotalAmount,
		query.Currency,
		query.InvoicePayload,
	)
	if err != nil {
		return domain.WrapBusinessError(fmt.Errorf("failed to handle pre_checkout_query: %w", err))
	}

	if !confirmed {
		s.Log.Info("pre_checkout_query rejected",
			"query_id", query.ID,
			"user_id", user.ID,
		)
		return nil // платёж отклонён, но это не ошибка
	}

	s.Log.Info("pre_checkout_query confirmed",
		"query_id", query.ID,
		"user_id", user.ID,
		"amount", query.TotalAmount,
	)

	return nil
}

// HandleSuccessfulPayment обрабатывает successful_payment от Telegram (для платежей Stars)
func (s *Service) HandleSuccessfulPayment(ctx context.Context, botID domain.BotId, message *domain.Message) error {
	if message == nil || message.SuccessfulPayment == nil {
		s.Log.Error("message or successful_payment is nil")
		return fmt.Errorf("invalid successful_payment")
	}

	if message.From == nil {
		s.Log.Error("message has no from")
		return fmt.Errorf("message has no from")
	}

	if message.Chat == nil {
		s.Log.Error("message has no chat")
		return fmt.Errorf("message has no chat")
	}

	botType, err := s.GetBotType(botID)
	if err != nil {
		return fmt.Errorf("failed to get bot_type for bot_id %s: %w", botID, err)
	}

	botService, ok := s.BotTypeToUsecase[botType]
	if !ok {
		return fmt.Errorf("unknown bot_type: %s", botType)
	}

	// Получаем или создаём пользователя
	user, err := botService.GetOrCreateUser(ctx, botID, message.From, message.Chat)
	if err != nil {
		return domain.WrapBusinessError(fmt.Errorf("failed to get or create user: %w", err))
	}

	// Парсим payment_id из payload
	paymentID, err := uuid.Parse(message.SuccessfulPayment.InvoicePayload)
	if err != nil {
		s.Log.Error("failed to parse payment_id from payload",
			"error", err,
			"payload", message.SuccessfulPayment.InvoicePayload,
		)
		return domain.WrapBusinessError(fmt.Errorf("invalid payment_id in payload: %w", err))
	}

	if s.PaymentUseCase == nil {
		s.Log.Error("payment use case not configured, cannot process successful_payment",
			"payment_id", paymentID,
		)
		return fmt.Errorf("payment use case not configured")
	}

	// Вызываем payment use case для обработки успешного платежа
	if err := s.PaymentUseCase.HandleSuccessfulPayment(
		ctx,
		botID,
		user.ID,
		message.Chat.ID,
		paymentID,
		message.SuccessfulPayment.TelegramPaymentChargeID,
	); err != nil {
		return domain.WrapBusinessError(fmt.Errorf("failed to handle successful_payment: %w", err))
	}

	s.Log.Info("successful_payment processed",
		"payment_id", paymentID,
		"user_id", user.ID,
		"amount", message.SuccessfulPayment.TotalAmount,
	)

	return nil
}
