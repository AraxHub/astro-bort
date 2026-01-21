package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
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
	StatusRepo      repository.IStatusRepo
	PaymentProvider paymentPort.IPaymentProvider // Telegram Stars провайдер
	TelegramService service.ITelegramService
	AlerterService  service.IAlerterService
	Log             *slog.Logger
}

func New(
	paymentRepo repository.IPaymentRepo,
	userRepo repository.IUserRepo,
	statusRepo repository.IStatusRepo,
	paymentProvider paymentPort.IPaymentProvider,
	telegramService service.ITelegramService,
	alerterService service.IAlerterService,
	log *slog.Logger,
) *Service {
	return &Service{
		PaymentRepo:     paymentRepo,
		UserRepo:        userRepo,
		StatusRepo:      statusRepo,
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
		// Логируем ошибку создания платежа
		errMsg := fmt.Sprintf("failed to create payment: %v", err)
		status := &domain.Status{
			ID:           uuid.New(),
			ObjectType:   domain.ObjectTypePayment,
			ObjectID:     paymentID,
			Status:       domain.StatusStatus(domain.PaymentError),
			ErrorMessage: &errMsg,
			Metadata: domain.BuildPaymentErrorMetadata(
				domain.StageCreatePayment,
				"CREATE_PAYMENT_ERROR",
				string(botID),
				map[string]interface{}{
					"user_id":    userID.String(),
					"product_id": productID,
					"amount":     amount,
				},
			),
			CreatedAt: time.Now(),
		}
		s.createOrLogStatus(ctx, status)
		s.sendAlertOrLog(ctx, status)
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}

	// Логируем создание платежа (промежуточная операция, без алерта)
	statusCreated := &domain.Status{
		ID:         uuid.New(),
		ObjectType: domain.ObjectTypePayment,
		ObjectID:   paymentID,
		Status:     domain.StatusStatus(domain.PaymentCreated),
		Metadata: domain.BuildPaymentSuccessMetadata(
			domain.StageCreatePayment,
			string(botID),
			map[string]interface{}{
				"user_id":    userID.String(),
				"product_id": productID,
				"amount":     amount,
			},
		),
		CreatedAt: time.Now(),
	}
	s.createOrLogStatus(ctx, statusCreated)

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

		// Логируем ошибку создания invoice
		errMsg := fmt.Sprintf("failed to create invoice: %v", err)
		status := &domain.Status{
			ID:           uuid.New(),
			ObjectType:   domain.ObjectTypePayment,
			ObjectID:     paymentID,
			Status:       domain.StatusStatus(domain.PaymentError),
			ErrorMessage: &errMsg,
			Metadata: domain.BuildPaymentErrorMetadata(
				domain.StageCreateInvoice,
				"CREATE_INVOICE_ERROR",
				string(botID),
				map[string]interface{}{
					"user_id":    userID.String(),
					"product_id": productID,
					"amount":     amount,
				},
			),
			CreatedAt: time.Now(),
		}
		s.createOrLogStatus(ctx, status)
		s.sendAlertOrLog(ctx, status)
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
			// Логируем ошибку отклонения
			errMsgLog := fmt.Sprintf("failed to reject pre_checkout_query: %v", err)
			status := &domain.Status{
				ID:           uuid.New(),
				ObjectType:   domain.ObjectTypePayment,
				ObjectID:     uuid.Nil, // payment не найден
				Status:       domain.StatusStatus(domain.PaymentError),
				ErrorMessage: &errMsgLog,
				Metadata: domain.BuildPaymentErrorMetadata(
					domain.StagePreCheckoutValidation,
					"PRE_CHECKOUT_REJECT_ERROR",
					string(botID),
					map[string]interface{}{
						"query_id": queryID,
						"payload":  payload,
						"user_id":  userID.String(),
					},
				),
				CreatedAt: time.Now(),
			}
			s.createOrLogStatus(ctx, status)
			s.sendAlertOrLog(ctx, status)
			return false, fmt.Errorf("failed to reject pre_checkout_query: %w", err)
		}

		// Логируем ошибку валидации (платёж не найден)
		errMsgLog := fmt.Sprintf("payment not found for payload: %v", err)
		status := &domain.Status{
			ID:           uuid.New(),
			ObjectType:   domain.ObjectTypePayment,
			ObjectID:     uuid.Nil, // payment не найден
			Status:       domain.StatusStatus(domain.PaymentError),
			ErrorMessage: &errMsgLog,
			Metadata: domain.BuildPaymentErrorMetadata(
				domain.StagePreCheckoutValidation,
				"PAYMENT_NOT_FOUND",
				string(botID),
				map[string]interface{}{
					"query_id": queryID,
					"payload":  payload,
					"user_id":  userID.String(),
				},
			),
			CreatedAt: time.Now(),
		}
		s.createOrLogStatus(ctx, status)
		s.sendAlertOrLog(ctx, status)
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
			errMsgLog := fmt.Sprintf("failed to reject pre_checkout_query: %v", err)
			status := &domain.Status{
				ID:           uuid.New(),
				ObjectType:   domain.ObjectTypePayment,
				ObjectID:     payment.ID,
				Status:       domain.StatusStatus(domain.PaymentError),
				ErrorMessage: &errMsgLog,
				Metadata: domain.BuildPaymentErrorMetadata(
					domain.StagePreCheckoutValidation,
					"PRE_CHECKOUT_REJECT_ERROR",
					string(botID),
					map[string]interface{}{
						"query_id":        queryID,
						"payment_user_id": payment.UserID.String(),
						"query_user_id":   userID.String(),
					},
				),
				CreatedAt: time.Now(),
			}
			s.createOrLogStatus(ctx, status)
			s.sendAlertOrLog(ctx, status)
			return false, fmt.Errorf("failed to reject pre_checkout_query: %w", err)
		}

		errMsgLog := "payment user mismatch"
		status := &domain.Status{
			ID:           uuid.New(),
			ObjectType:   domain.ObjectTypePayment,
			ObjectID:     payment.ID,
			Status:       domain.StatusStatus(domain.PaymentError),
			ErrorMessage: &errMsgLog,
			Metadata: domain.BuildPaymentErrorMetadata(
				domain.StagePreCheckoutValidation,
				"USER_MISMATCH",
				string(botID),
				map[string]interface{}{
					"query_id":        queryID,
					"payment_user_id": payment.UserID.String(),
					"query_user_id":   userID.String(),
				},
			),
			CreatedAt: time.Now(),
		}
		s.createOrLogStatus(ctx, status)
		s.sendAlertOrLog(ctx, status)
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
			errMsgLog := fmt.Sprintf("failed to reject pre_checkout_query: %v", err)
			status := &domain.Status{
				ID:           uuid.New(),
				ObjectType:   domain.ObjectTypePayment,
				ObjectID:     payment.ID,
				Status:       domain.StatusStatus(domain.PaymentError),
				ErrorMessage: &errMsgLog,
				Metadata: domain.BuildPaymentErrorMetadata(
					domain.StagePreCheckoutValidation,
					"PRE_CHECKOUT_REJECT_ERROR",
					string(botID),
					map[string]interface{}{
						"query_id":       queryID,
						"payment_amount": payment.Amount,
						"query_amount":   amount,
					},
				),
				CreatedAt: time.Now(),
			}
			s.createOrLogStatus(ctx, status)
			s.sendAlertOrLog(ctx, status)
			return false, fmt.Errorf("failed to reject pre_checkout_query: %w", err)
		}

		errMsgLog := fmt.Sprintf("payment amount mismatch: expected %d, got %d", payment.Amount, amount)
		status := &domain.Status{
			ID:           uuid.New(),
			ObjectType:   domain.ObjectTypePayment,
			ObjectID:     payment.ID,
			Status:       domain.StatusStatus(domain.PaymentError),
			ErrorMessage: &errMsgLog,
			Metadata: domain.BuildPaymentErrorMetadata(
				domain.StagePreCheckoutValidation,
				"AMOUNT_MISMATCH",
				string(botID),
				map[string]interface{}{
					"query_id":       queryID,
					"payment_amount": payment.Amount,
					"query_amount":   amount,
				},
			),
			CreatedAt: time.Now(),
		}
		s.createOrLogStatus(ctx, status)
		s.sendAlertOrLog(ctx, status)
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
			errMsgLog := fmt.Sprintf("failed to reject pre_checkout_query: %v", err)
			status := &domain.Status{
				ID:           uuid.New(),
				ObjectType:   domain.ObjectTypePayment,
				ObjectID:     payment.ID,
				Status:       domain.StatusStatus(domain.PaymentError),
				ErrorMessage: &errMsgLog,
				Metadata: domain.BuildPaymentErrorMetadata(
					domain.StagePreCheckoutValidation,
					"PRE_CHECKOUT_REJECT_ERROR",
					string(botID),
					map[string]interface{}{
						"query_id":         queryID,
						"payment_currency": payment.Currency,
						"query_currency":   currency,
					},
				),
				CreatedAt: time.Now(),
			}
			s.createOrLogStatus(ctx, status)
			s.sendAlertOrLog(ctx, status)
			return false, fmt.Errorf("failed to reject pre_checkout_query: %w", err)
		}

		errMsgLog := fmt.Sprintf("payment currency mismatch: expected %s, got %s", payment.Currency, currency)
		status := &domain.Status{
			ID:           uuid.New(),
			ObjectType:   domain.ObjectTypePayment,
			ObjectID:     payment.ID,
			Status:       domain.StatusStatus(domain.PaymentError),
			ErrorMessage: &errMsgLog,
			Metadata: domain.BuildPaymentErrorMetadata(
				domain.StagePreCheckoutValidation,
				"CURRENCY_MISMATCH",
				string(botID),
				map[string]interface{}{
					"query_id":         queryID,
					"payment_currency": payment.Currency,
					"query_currency":   currency,
				},
			),
			CreatedAt: time.Now(),
		}
		s.createOrLogStatus(ctx, status)
		s.sendAlertOrLog(ctx, status)
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
			errMsgLog := fmt.Sprintf("failed to reject pre_checkout_query: %v", err)
			status := &domain.Status{
				ID:           uuid.New(),
				ObjectType:   domain.ObjectTypePayment,
				ObjectID:     payment.ID,
				Status:       domain.StatusStatus(domain.PaymentError),
				ErrorMessage: &errMsgLog,
				Metadata: domain.BuildPaymentErrorMetadata(
					domain.StagePreCheckoutValidation,
					"PRE_CHECKOUT_REJECT_ERROR",
					string(botID),
					map[string]interface{}{
						"query_id": queryID,
						"status":   payment.Status,
					},
				),
				CreatedAt: time.Now(),
			}
			s.createOrLogStatus(ctx, status)
			s.sendAlertOrLog(ctx, status)
			return false, fmt.Errorf("failed to reject pre_checkout_query: %w", err)
		}

		errMsgLog := fmt.Sprintf("payment already processed with status: %s", payment.Status)
		status := &domain.Status{
			ID:           uuid.New(),
			ObjectType:   domain.ObjectTypePayment,
			ObjectID:     payment.ID,
			Status:       domain.StatusStatus(domain.PaymentError),
			ErrorMessage: &errMsgLog,
			Metadata: domain.BuildPaymentErrorMetadata(
				domain.StagePreCheckoutValidation,
				"ALREADY_PROCESSED",
				string(botID),
				map[string]interface{}{
					"query_id": queryID,
					"status":   payment.Status,
				},
			),
			CreatedAt: time.Now(),
		}
		s.createOrLogStatus(ctx, status)
		s.sendAlertOrLog(ctx, status)
		return false, nil
	}

	// Все проверки пройдены - подтверждаем платёж
	if err := s.PaymentProvider.ConfirmPreCheckout(ctx, string(botID), queryID, true, nil); err != nil {
		errMsgLog := fmt.Sprintf("failed to confirm pre_checkout_query: %v", err)
		status := &domain.Status{
			ID:           uuid.New(),
			ObjectType:   domain.ObjectTypePayment,
			ObjectID:     payment.ID,
			Status:       domain.StatusStatus(domain.PaymentError),
			ErrorMessage: &errMsgLog,
			Metadata: domain.BuildPaymentErrorMetadata(
				domain.StagePreCheckoutValidation,
				"PRE_CHECKOUT_CONFIRM_ERROR",
				string(botID),
				map[string]interface{}{
					"query_id": queryID,
					"user_id":  userID.String(),
				},
			),
			CreatedAt: time.Now(),
		}
		s.createOrLogStatus(ctx, status)
		s.sendAlertOrLog(ctx, status)
		return false, fmt.Errorf("failed to confirm pre_checkout_query: %w", err)
	}

	s.Log.Info("pre_checkout_query confirmed",
		"query_id", queryID,
		"payment_id", payment.ID,
		"user_id", userID,
	)

	// Логируем успешную валидацию (промежуточная операция, без алерта)
	statusValidated := &domain.Status{
		ID:         uuid.New(),
		ObjectType: domain.ObjectTypePayment,
		ObjectID:   payment.ID,
		Status:     domain.StatusStatus(domain.PaymentCreated), // остаётся в статусе created до успешной оплаты
		Metadata: domain.BuildPaymentSuccessMetadata(
			domain.StagePreCheckoutValidation,
			string(botID),
			map[string]interface{}{
				"query_id": queryID,
				"user_id":  userID.String(),
			},
		),
		CreatedAt: time.Now(),
	}
	s.createOrLogStatus(ctx, statusValidated)

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
		errMsgLog := fmt.Sprintf("failed to get payment: %v", err)
		status := &domain.Status{
			ID:           uuid.New(),
			ObjectType:   domain.ObjectTypePayment,
			ObjectID:     paymentID,
			Status:       domain.StatusStatus(domain.PaymentError),
			ErrorMessage: &errMsgLog,
			Metadata: domain.BuildPaymentErrorMetadata(
				domain.StageUpdateStatus,
				"GET_PAYMENT_ERROR",
				string(botID),
				map[string]interface{}{
					"user_id": userID.String(),
				},
			),
			CreatedAt: time.Now(),
		}
		s.createOrLogStatus(ctx, status)
		s.sendAlertOrLog(ctx, status)
		return fmt.Errorf("failed to get payment: %w", err)
	}

	// Проверяем, что платёж принадлежит пользователю
	if payment.UserID != userID {
		errMsgLog := fmt.Sprintf("payment user mismatch: payment belongs to %s, but user is %s", payment.UserID, userID)
		status := &domain.Status{
			ID:           uuid.New(),
			ObjectType:   domain.ObjectTypePayment,
			ObjectID:     paymentID,
			Status:       domain.StatusStatus(domain.PaymentError),
			ErrorMessage: &errMsgLog,
			Metadata: domain.BuildPaymentErrorMetadata(
				domain.StageUpdateStatus,
				"USER_MISMATCH",
				string(botID),
				map[string]interface{}{
					"payment_user_id": payment.UserID.String(),
					"query_user_id":   userID.String(),
				},
			),
			CreatedAt: time.Now(),
		}
		s.createOrLogStatus(ctx, status)
		s.sendAlertOrLog(ctx, status)
		return fmt.Errorf("payment user mismatch: payment belongs to %s, but user is %s", payment.UserID, userID)
	}

	// Проверяем статус (должен быть pending)
	if payment.Status != domain.PaymentStatusPending {
		s.Log.Warn("payment already processed",
			"payment_id", paymentID,
			"status", payment.Status,
		)
		// Логируем, но не алертим (это нормальная ситуация - идемпотентность)
		status := &domain.Status{
			ID:         uuid.New(),
			ObjectType: domain.ObjectTypePayment,
			ObjectID:   paymentID,
			Status:     domain.StatusStatus(domain.PaymentSucceeded), // уже обработан
			Metadata: domain.BuildPaymentSuccessMetadata(
				domain.StageUpdateStatus,
				string(botID),
				map[string]interface{}{
					"user_id":    userID.String(),
					"product_id": payment.ProductID,
					"amount":     payment.Amount,
					"note":       "already processed",
				},
			),
			CreatedAt: time.Now(),
		}
		s.createOrLogStatus(ctx, status)
		// Не алертим - это нормальная ситуация
		return nil // уже обработан, ничего не делаем
	}

	// Обновляем статус на succeeded
	now := time.Now()
	if err := s.PaymentRepo.UpdateStatus(ctx, paymentID, domain.PaymentStatusSucceeded, &now, nil, nil); err != nil {
		errMsgLog := fmt.Sprintf("failed to update payment status: %v", err)
		status := &domain.Status{
			ID:           uuid.New(),
			ObjectType:   domain.ObjectTypePayment,
			ObjectID:     paymentID,
			Status:       domain.StatusStatus(domain.PaymentError),
			ErrorMessage: &errMsgLog,
			Metadata: domain.BuildPaymentErrorMetadata(
				domain.StageUpdateStatus,
				"UPDATE_STATUS_ERROR",
				string(botID),
				map[string]interface{}{
					"user_id":    userID.String(),
					"product_id": payment.ProductID,
					"amount":     payment.Amount,
				},
			),
			CreatedAt: time.Now(),
		}
		s.createOrLogStatus(ctx, status)
		s.sendAlertOrLog(ctx, status)
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

		// Логируем критическую ошибку (деньги списаны, но продукт не выдан)
		errMsgLog := fmt.Sprintf("failed to grant product after payment: %v", err)
		status := &domain.Status{
			ID:           uuid.New(),
			ObjectType:   domain.ObjectTypePayment,
			ObjectID:     paymentID,
			Status:       domain.StatusStatus(domain.PaymentError),
			ErrorMessage: &errMsgLog,
			Metadata: domain.BuildPaymentErrorMetadata(
				domain.StageGrantProduct,
				"GRANT_PRODUCT_ERROR",
				string(botID),
				map[string]interface{}{
					"user_id":    userID.String(),
					"product_id": payment.ProductID,
					"amount":     payment.Amount,
				},
			),
			CreatedAt: time.Now(),
		}
		s.createOrLogStatus(ctx, status)
		s.sendAlertOrLog(ctx, status)
	}

	// Логируем успешное получение денег (алертим только это)
	statusSucceeded := &domain.Status{
		ID:         uuid.New(),
		ObjectType: domain.ObjectTypePayment,
		ObjectID:   paymentID,
		Status:     domain.StatusStatus(domain.PaymentSucceeded),
		Metadata: domain.BuildPaymentSuccessMetadata(
			domain.StageUpdateStatus,
			string(botID),
			map[string]interface{}{
				"user_id":    userID.String(),
				"product_id": payment.ProductID,
				"amount":     payment.Amount,
				"charge_id":  telegramPaymentChargeID,
			},
		),
		CreatedAt: time.Now(),
	}
	s.createOrLogStatus(ctx, statusSucceeded)
	s.sendAlertOrLog(ctx, statusSucceeded)

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

// createOrLogStatus создаёт статус в БД, не падает если репозиторий не настроен
func (s *Service) createOrLogStatus(ctx context.Context, status *domain.Status) {
	if s.StatusRepo == nil {
		return
	}

	if err := s.StatusRepo.Create(ctx, status); err != nil {
		s.Log.Warn("failed to create status (non-critical)",
			"error", err,
			"object_type", status.ObjectType,
			"object_id", status.ObjectID,
			"status", status.Status,
		)
	}
}

// sendAlertOrLog отправляет алерт в Telegram канал, не падает если алертер не настроен
func (s *Service) sendAlertOrLog(ctx context.Context, status *domain.Status) {
	if s.AlerterService == nil {
		return
	}

	message := s.formatPaymentAlertMessage(status)
	if message == "" {
		return
	}

	if err := s.AlerterService.SendAlert(ctx, message); err != nil {
		s.Log.Warn("failed to send alert (non-critical)",
			"error", err,
			"object_id", status.ObjectID,
			"status", status.Status,
		)
	}
}

// formatPaymentAlertMessage форматирует сообщение для алерта на основе статуса платежа
func (s *Service) formatPaymentAlertMessage(status *domain.Status) string {
	var builder strings.Builder
	paymentID := status.ObjectID.String()

	switch domain.PaymentStatusEnum(status.Status) {
	case domain.PaymentSucceeded:
		// Успешный алерт только на получение денег
		builder.WriteString("✅ *Payment Succeeded*\n\n")
		builder.WriteString(fmt.Sprintf("*Payment ID:* `%s`\n", paymentID))

		// Добавляем контекст из metadata если есть
		if len(status.Metadata) > 0 {
			var metadata map[string]interface{}
			if err := json.Unmarshal(status.Metadata, &metadata); err == nil {
				if userID, ok := metadata["user_id"].(string); ok {
					builder.WriteString(fmt.Sprintf("*User ID:* `%s`\n", userID))
				}
				if amount, ok := metadata["amount"].(float64); ok {
					builder.WriteString(fmt.Sprintf("*Amount:* %.0f XTR\n", amount))
				}
				if productID, ok := metadata["product_id"].(string); ok {
					builder.WriteString(fmt.Sprintf("*Product ID:* `%s`\n", productID))
				}
			}
		}

	case domain.PaymentError:
		// Ошибки алертим всегда с @members
		builder.WriteString(fmt.Sprintf("❌ *Payment Error*\n\n%s\n", "@nhoj41_3 @matarseks @romanovnl"))
		builder.WriteString(fmt.Sprintf("*Payment ID:* `%s`\n", paymentID))

		// Определяем этап из metadata
		if len(status.Metadata) > 0 {
			var metadata map[string]interface{}
			if err := json.Unmarshal(status.Metadata, &metadata); err == nil {
				if stage, ok := metadata["stage"].(string); ok {
					builder.WriteString(fmt.Sprintf("*Stage:* `%s`\n", stage))
				}
				if phase, ok := metadata["phase"].(string); ok {
					builder.WriteString(fmt.Sprintf("*Phase:* `%s`\n", phase))
				}
			}
		}

		// Сообщение об ошибке
		if status.ErrorMessage != nil {
			errMsg := *status.ErrorMessage
			builder.WriteString(fmt.Sprintf("*Error:* %s\n", errMsg))
		}

	default:
		// Промежуточные операции не алертим
		return ""
	}

	return builder.String()
}
