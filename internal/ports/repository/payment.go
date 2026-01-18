package repository

import (
	"context"
	"time"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/google/uuid"
)

// IPaymentRepo интерфейс для работы с платежами в БД
type IPaymentRepo interface {
	Create(ctx context.Context, payment *domain.Payment) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Payment, error)
	GetByProviderID(ctx context.Context, providerID string) (*domain.Payment, error)
	GetByPayload(ctx context.Context, payload string) (*domain.Payment, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.PaymentStatus, succeededAt, failedAt *time.Time, errorMessage *string) error
	GetLastSuccessfulPaymentDate(ctx context.Context, userID uuid.UUID) (*time.Time, error)
	GetBotIDForUser(ctx context.Context, userID uuid.UUID) (string, error)
}
