package payment

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	paymentPort "github.com/admin/tg-bots/astro-bot/internal/ports/payment"
	"github.com/admin/tg-bots/astro-bot/internal/ports/repository"
	"github.com/admin/tg-bots/astro-bot/internal/ports/service"
	"github.com/google/uuid"
)

type Service struct {
	PaymentRepo     repository.IPaymentRepo
	UserRepo        repository.IUserRepo
	PaymentProvider paymentPort.IPaymentProvider // Telegram Stars провайдер
	TelegramService service.ITelegramService
	AlerterService  service.IAlerterService
	Log             *slog.Logger
}

func New(
	paymentRepo repository.IPaymentRepo,
	userRepo repository.IUserRepo,
	paymentProvider paymentPort.IPaymentProvider,
	telegramService service.ITelegramService,
	alerterService service.IAlerterService,
	log *slog.Logger,
) *Service {
	return &Service{
		PaymentRepo:     paymentRepo,
		UserRepo:        userRepo,
		PaymentProvider: paymentProvider,
		TelegramService: telegramService,
		AlerterService:  alerterService,
		Log:             log,
	}
}

func (s *Service) CreatePayment(
	ctx context.Context,
	botID domain.BotId,
	userID uuid.UUID,
	chatID int64,
	productID string,
	productTitle string,
	description string,
	amount int64, // количество звёзд
) (*domain.Payment, error) {
	paymentID := uuid.New()
	now := time.Now()

	payment := &domain.Payment{
		ID:           paymentID,
		UserID:       userID,
		BotID:        botID,
		Amount:       amount,
		Currency:     "XTR", // Telegram Stars
		Method:       domain.PaymentMethodTelegramStars,
		ProviderID:   "", // будет заполнен после отправки invoice
		Status:       domain.PaymentStatusPending,
		ProductID:    productID,
		ProductTitle: productTitle,
		Metadata: map[string]interface{}{
			"payload": paymentID.String(), // для поиска по payload
		},
		CreatedAt: now,
	}

	if err := s.PaymentRepo.Create(ctx, payment); err != nil {
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	// Создаём invoice через провайдер
	invoiceReq := paymentPort.CreateInvoiceRequest{
		BotID:        string(botID),
		UserID:       userID,
		ChatID:       chatID,
		ProductID:    productID,
		ProductTitle: productTitle,
		Description:  description,
		Amount:       amount,
		Currency:     "XTR",
		Payload:      paymentID.String(), // используем payment_id как payload
	}

	invoiceResult, err := s.PaymentProvider.CreateInvoice(ctx, invoiceReq)
	if err != nil {
		// Обновляем статус на failed
		now := time.Now()
		_ = s.PaymentRepo.UpdateStatus(ctx, paymentID, domain.PaymentStatusFailed, nil, &now, stringPtr("failed to create invoice"))
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	// Обновляем provider_id (invoice_id)
	payment.ProviderID = invoiceResult.InvoiceID
	// Обновляем в БД через UpdateStatus (или можно добавить отдельный метод UpdateProviderID)
	// Пока оставим как есть, provider_id можно обновить позже при необходимости

	s.Log.Info("payment created and invoice sent",
		"payment_id", paymentID,
		"user_id", userID,
		"product_id", productID,
		"amount", amount,
	)

	return payment, nil
}

// HandlePreCheckoutQuery обрабатывает pre_checkout_query от Telegram
// Возвращает true если платёж подтверждён, false если отклонён
func (s *Service) HandlePreCheckoutQuery(
	ctx context.Context,
	botID domain.BotId,
	queryID string,
	userID uuid.UUID,
	amount int64,
	currency string,
	payload string,
) (bool, error) {
	// Находим платёж по payload
	payment, err := s.PaymentRepo.GetByPayload(ctx, payload)
	if err != nil {
		s.Log.Warn("payment not found for pre_checkout_query",
			"query_id", queryID,
			"payload", payload,
			"error", err,
		)
		// Отклоняем платёж
		errorMsg := "Платёж не найден"
		if err := s.PaymentProvider.ConfirmPreCheckout(ctx, string(botID), queryID, false, &errorMsg); err != nil {
			return false, fmt.Errorf("failed to reject pre_checkout_query: %w", err)
		}
		return false, nil
	}

	// Проверяем, что платёж принадлежит пользователю
	if payment.UserID != userID {
		s.Log.Warn("payment user mismatch",
			"query_id", queryID,
			"payment_id", payment.ID,
			"payment_user_id", payment.UserID,
			"query_user_id", userID,
		)
		errorMsg := "Платёж не принадлежит вам"
		if err := s.PaymentProvider.ConfirmPreCheckout(ctx, string(botID), queryID, false, &errorMsg); err != nil {
			return false, fmt.Errorf("failed to reject pre_checkout_query: %w", err)
		}
		return false, nil
	}

	// Проверяем сумму
	if payment.Amount != amount {
		s.Log.Warn("payment amount mismatch",
			"query_id", queryID,
			"payment_id", payment.ID,
			"payment_amount", payment.Amount,
			"query_amount", amount,
		)
		errorMsg := "Сумма платежа не совпадает"
		if err := s.PaymentProvider.ConfirmPreCheckout(ctx, string(botID), queryID, false, &errorMsg); err != nil {
			return false, fmt.Errorf("failed to reject pre_checkout_query: %w", err)
		}
		return false, nil
	}

	// Проверяем валюту
	if payment.Currency != currency {
		s.Log.Warn("payment currency mismatch",
			"query_id", queryID,
			"payment_id", payment.ID,
			"payment_currency", payment.Currency,
			"query_currency", currency,
		)
		errorMsg := "Валюта платежа не совпадает"
		if err := s.PaymentProvider.ConfirmPreCheckout(ctx, string(botID), queryID, false, &errorMsg); err != nil {
			return false, fmt.Errorf("failed to reject pre_checkout_query: %w", err)
		}
		return false, nil
	}

	// Проверяем статус (должен быть pending)
	if payment.Status != domain.PaymentStatusPending {
		s.Log.Warn("payment already processed",
			"query_id", queryID,
			"payment_id", payment.ID,
			"status", payment.Status,
		)
		errorMsg := "Платёж уже обработан"
		if err := s.PaymentProvider.ConfirmPreCheckout(ctx, string(botID), queryID, false, &errorMsg); err != nil {
			return false, fmt.Errorf("failed to reject pre_checkout_query: %w", err)
		}
		return false, nil
	}

	// Все проверки пройдены - подтверждаем платёж
	if err := s.PaymentProvider.ConfirmPreCheckout(ctx, string(botID), queryID, true, nil); err != nil {
		return false, fmt.Errorf("failed to confirm pre_checkout_query: %w", err)
	}

	s.Log.Info("pre_checkout_query confirmed",
		"query_id", queryID,
		"payment_id", payment.ID,
		"user_id", userID,
	)

	return true, nil
}

// HandleSuccessfulPayment обрабатывает успешный платёж (выдаёт продукт, уведомляет пользователя)
func (s *Service) HandleSuccessfulPayment(
	ctx context.Context,
	botID domain.BotId,
	userID uuid.UUID,
	chatID int64,
	paymentID uuid.UUID,
	telegramPaymentChargeID string,
) error {
	// Получаем платёж
	payment, err := s.PaymentRepo.GetByID(ctx, paymentID)
	if err != nil {
		return fmt.Errorf("failed to get payment: %w", err)
	}

	// Проверяем, что платёж принадлежит пользователю
	if payment.UserID != userID {
		return fmt.Errorf("payment user mismatch: payment belongs to %s, but user is %s", payment.UserID, userID)
	}

	// Проверяем статус (должен быть pending)
	if payment.Status != domain.PaymentStatusPending {
		s.Log.Warn("payment already processed",
			"payment_id", paymentID,
			"status", payment.Status,
		)
		return nil // уже обработан, ничего не делаем
	}

	// Обновляем статус на succeeded
	now := time.Now()
	if err := s.PaymentRepo.UpdateStatus(ctx, paymentID, domain.PaymentStatusSucceeded, &now, nil, nil); err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	// Выдаём продукт (бизнес-логика)
	if err := s.grantProduct(ctx, botID, userID, payment.ProductID); err != nil {
		// Логируем ошибку, но не откатываем платёж (деньги уже списаны)
		s.Log.Error("failed to grant product after payment",
			"error", err,
			"payment_id", paymentID,
			"user_id", userID,
			"product_id", payment.ProductID,
		)

		// Отправляем алерт
		if s.AlerterService != nil {
			alertMsg := fmt.Sprintf("⚠️ *Payment Success, Product Grant Failed*\n\n*Payment ID:* %s\n*User ID:* %s\n*Product ID:* %s\n*Error:* %s",
				paymentID, userID, payment.ProductID, err.Error())
			_ = s.AlerterService.SendAlert(ctx, alertMsg)
		}
	}

	// Уведомляем пользователя об успешной оплате
	message := fmt.Sprintf("✅ Платёж успешно обработан!\n\nВы приобрели: %s", payment.ProductTitle)
	if err := s.TelegramService.SendMessage(ctx, botID, chatID, message); err != nil {
		s.Log.Warn("failed to send payment success notification",
			"error", err,
			"payment_id", paymentID,
			"chat_id", chatID,
		)
	}

	s.Log.Info("payment processed successfully",
		"payment_id", paymentID,
		"user_id", userID,
		"product_id", payment.ProductID,
		"amount", payment.Amount,
	)

	return nil
}

// grantProduct выдаёт продукт пользователю (бизнес-логика)
func (s *Service) grantProduct(ctx context.Context, botID domain.BotId, userID uuid.UUID, productID string) error {
	if err := s.UserRepo.SetPaidStatus(ctx, userID, true); err != nil {
		return fmt.Errorf("failed to set paid status: %w", err)
	}

	s.Log.Info("product granted",
		"user_id", userID,
		"product_id", productID,
	)

	return nil
}

func stringPtr(s string) *string {
	return &s
}
