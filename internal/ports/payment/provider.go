package payment

import (
	"context"

	"github.com/google/uuid"
)

// IPaymentProvider интерфейс для платёжного провайдера (Telegram Stars, YooKassa и т.д.)
// Use case зависит только от этого интерфейса, не зная деталей реализации
type IPaymentProvider interface {
	// CreateInvoice создаёт invoice для отправки пользователю для Telegram Stars возвращает данные для sendInvoice
	CreateInvoice(ctx context.Context, req CreateInvoiceRequest) (*CreateInvoiceResult, error)

	// ConfirmPreCheckout подтверждает pre_checkout_query (для Telegram Stars)
	// Возвращает true если подтверждено, false если отклонено
	// botID нужен для выбора правильного Telegram client
	ConfirmPreCheckout(ctx context.Context, botID string, queryID string, ok bool, errorMessage *string) error
}

// CreateInvoiceRequest запрос на создание invoice
type CreateInvoiceRequest struct {
	BotID                     string
	UserID                    uuid.UUID
	ChatID                    int64
	ProductID                 string
	ProductTitle              string
	Description               string
	Amount                    int64  // количество звёзд
	Currency                  string // "XTR" для Stars
	Payload                   string // уникальный payload для идентификации платежа (обычно payment_id)
	PhotoURL                  *string
	PhotoSize                 *int64
	PhotoWidth                *int64
	PhotoHeight               *int64
	NeedName                  bool
	NeedPhoneNumber           bool
	NeedEmail                 bool
	NeedShippingAddress       bool
	SendPhoneNumberToProvider bool
	SendEmailToProvider       bool
	IsFlexible                bool
}

// CreateInvoiceResult результат создания invoice
type CreateInvoiceResult struct {
	InvoiceID string // ID invoice в системе провайдера (для Telegram Stars - это будет message_id после sendInvoice)
	// Для других провайдеров здесь может быть confirmation_url и т.д.
}
