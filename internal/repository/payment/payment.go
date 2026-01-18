package paymentRepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	ports "github.com/admin/tg-bots/astro-bot/internal/ports/repository"

	"log/slog"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/admin/tg-bots/astro-bot/internal/ports/persistence"
	"github.com/google/uuid"
)

type paymentColumns struct {
	TableName    string
	ID           string
	UserID       string
	BotID        string
	Amount       string
	Currency     string
	Method       string
	ProviderID   string
	Status       string
	ProductID    string
	ProductTitle string
	Metadata     string
	CreatedAt    string
	SucceededAt  string
	FailedAt     string
	ErrorMessage string
}

type Repository struct {
	db      persistence.Persistence
	Log     *slog.Logger
	columns paymentColumns
}

// New создаёт новый репозиторий для работы с платежами
func New(db persistence.Persistence, log *slog.Logger) ports.IPaymentRepo {
	cols := paymentColumns{
		TableName:    "payments",
		ID:           "id",
		UserID:       "user_id",
		BotID:        "bot_id",
		Amount:       "amount",
		Currency:     "currency",
		Method:       "method",
		ProviderID:   "provider_id",
		Status:       "status",
		ProductID:    "product_id",
		ProductTitle: "product_title",
		Metadata:     "metadata",
		CreatedAt:    "created_at",
		SucceededAt:  "succeeded_at",
		FailedAt:     "failed_at",
		ErrorMessage: "error_message",
	}
	return &Repository{
		db:      db,
		Log:     log,
		columns: cols,
	}
}

// allColumns возвращает строку со всеми колонками (15 полей)
func (r *Repository) allColumns() string {
	return fmt.Sprintf("%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s",
		r.columns.ID,
		r.columns.UserID,
		r.columns.BotID,
		r.columns.Amount,
		r.columns.Currency,
		r.columns.Method,
		r.columns.ProviderID,
		r.columns.Status,
		r.columns.ProductID,
		r.columns.ProductTitle,
		r.columns.Metadata,
		r.columns.CreatedAt,
		r.columns.SucceededAt,
		r.columns.FailedAt,
		r.columns.ErrorMessage,
	)
}

// Create создаёт новый платёж
func (r *Repository) Create(ctx context.Context, payment *domain.Payment) error {
	// Сериализуем metadata через Value() (реализует driver.Valuer)
	metadataValue, err := payment.Metadata.Value()
	if err != nil {
		r.Log.Error("failed to marshal payment metadata",
			"error", err,
			"payment_id", payment.ID,
		)
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := fmt.Sprintf(`INSERT INTO %s (%s) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
		r.columns.TableName,
		r.allColumns(),
	)

	err = r.db.Exec(ctx, query,
		payment.ID,
		payment.UserID,
		string(payment.BotID),
		payment.Amount,
		payment.Currency,
		string(payment.Method),
		payment.ProviderID,
		string(payment.Status),
		payment.ProductID,
		payment.ProductTitle,
		metadataValue,
		payment.CreatedAt,
		payment.SucceededAt,
		payment.FailedAt,
		payment.ErrorMessage,
	)
	if err != nil {
		r.Log.Error("failed to create payment",
			"error", err,
			"payment_id", payment.ID,
			"user_id", payment.UserID,
		)
		return fmt.Errorf("failed to create payment: %w", err)
	}

	r.Log.Debug("payment created successfully",
		"payment_id", payment.ID,
		"user_id", payment.UserID,
		"amount", payment.Amount,
	)
	return nil
}

// GetByID получает платёж по ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Payment, error) {
	var payment domain.Payment

	query := fmt.Sprintf(`SELECT %s FROM %s WHERE %s = $1`,
		r.allColumns(),
		r.columns.TableName,
		r.columns.ID,
	)

	err := r.db.Get(ctx, &payment, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.Log.Warn("payment not found", "payment_id", id)
			return nil, fmt.Errorf("payment not found: %w", err)
		}
		r.Log.Error("failed to get payment",
			"error", err,
			"payment_id", id,
		)
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	r.Log.Debug("payment retrieved successfully", "payment_id", id)
	return &payment, nil
}

// GetByProviderID получает платёж по ID провайдера
func (r *Repository) GetByProviderID(ctx context.Context, providerID string) (*domain.Payment, error) {
	var payment domain.Payment

	query := fmt.Sprintf(`SELECT %s FROM %s WHERE %s = $1`,
		r.allColumns(),
		r.columns.TableName,
		r.columns.ProviderID,
	)

	err := r.db.Get(ctx, &payment, query, providerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.Log.Warn("payment not found by provider_id", "provider_id", providerID)
			return nil, fmt.Errorf("payment not found: %w", err)
		}
		r.Log.Error("failed to get payment by provider_id",
			"error", err,
			"provider_id", providerID,
		)
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	r.Log.Debug("payment retrieved successfully by provider_id", "provider_id", providerID)
	return &payment, nil
}

// GetByPayload получает платёж по payload (для Telegram Stars)
// payload хранится в metadata как "payload": "payment_id"
func (r *Repository) GetByPayload(ctx context.Context, payload string) (*domain.Payment, error) {
	var payment domain.Payment

	// Ищем по metadata->payload и статусу pending
	query := fmt.Sprintf(`SELECT %s FROM %s WHERE %s::jsonb->>'payload' = $1 AND %s = $2`,
		r.allColumns(),
		r.columns.TableName,
		r.columns.Metadata,
		r.columns.Status,
	)

	err := r.db.Get(ctx, &payment, query, payload, string(domain.PaymentStatusPending))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.Log.Warn("payment not found by payload", "payload", payload)
			return nil, fmt.Errorf("payment not found: %w", err)
		}
		r.Log.Error("failed to get payment by payload",
			"error", err,
			"payload", payload,
		)
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	r.Log.Debug("payment retrieved successfully by payload", "payload", payload)
	return &payment, nil
}

// UpdateStatus обновляет статус платежа
func (r *Repository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.PaymentStatus, succeededAt, failedAt *time.Time, errorMessage *string) error {
	query := fmt.Sprintf(`UPDATE %s SET %s = $1, %s = $2, %s = $3, %s = $4 WHERE %s = $5`,
		r.columns.TableName,
		r.columns.Status,
		r.columns.SucceededAt,
		r.columns.FailedAt,
		r.columns.ErrorMessage,
		r.columns.ID,
	)

	err := r.db.Exec(ctx, query, string(status), succeededAt, failedAt, errorMessage, id)
	if err != nil {
		r.Log.Error("failed to update payment status",
			"error", err,
			"payment_id", id,
			"status", status,
		)
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	r.Log.Debug("payment status updated successfully",
		"payment_id", id,
		"status", status,
	)
	return nil
}

// GetLastSuccessfulPaymentDate возвращает дату последнего успешного платежа для пользователя
func (r *Repository) GetLastSuccessfulPaymentDate(ctx context.Context, userID uuid.UUID) (*time.Time, error) {
	query := fmt.Sprintf(`
		SELECT MAX(%s) 
		FROM %s 
		WHERE %s = $1 
		  AND %s = $2 
		  AND %s IS NOT NULL
	`,
		r.columns.SucceededAt,
		r.columns.TableName,
		r.columns.UserID,
		r.columns.Status,
		r.columns.SucceededAt,
	)

	var succeededAt *time.Time
	err := r.db.Get(ctx, &succeededAt, query, userID, string(domain.PaymentStatusSucceeded))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Нет успешных платежей
		}
		r.Log.Error("failed to get last successful payment date",
			"error", err,
			"user_id", userID,
		)
		return nil, fmt.Errorf("failed to get last successful payment date: %w", err)
	}

	return succeededAt, nil
}

// GetBotIDForUser возвращает bot_id последнего успешного платежа пользователя
func (r *Repository) GetBotIDForUser(ctx context.Context, userID uuid.UUID) (string, error) {
	query := fmt.Sprintf(`
		SELECT %s 
		FROM %s 
		WHERE %s = $1 
		  AND %s = $2 
		  AND %s IS NOT NULL
		ORDER BY %s DESC
		LIMIT 1
	`,
		r.columns.BotID,
		r.columns.TableName,
		r.columns.UserID,
		r.columns.Status,
		r.columns.SucceededAt,
		r.columns.SucceededAt,
	)

	var botID string
	err := r.db.Get(ctx, &botID, query, userID, string(domain.PaymentStatusSucceeded))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("no successful payment found for user")
		}
		r.Log.Error("failed to get bot_id for user",
			"error", err,
			"user_id", userID,
		)
		return "", fmt.Errorf("failed to get bot_id for user: %w", err)
	}

	return botID, nil
}
