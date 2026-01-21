package usecase

import (
	"context"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/google/uuid"
)

// IPaymentUseCase интерфейс для работы с платежами (use case слой)
type IPaymentUseCase interface {
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
