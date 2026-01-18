package service

import (
	"context"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/google/uuid"
)

// IPaymentService интерфейс для работы с платежами (для использования в других use cases)
type IPaymentService interface {
	CreatePayment(
		ctx context.Context,
		botID domain.BotId,
		userID uuid.UUID,
		chatID int64,
		productID string,
		productTitle string,
		description string,
		amount int64, // количество звёзд
	) (*domain.Payment, error)
}
