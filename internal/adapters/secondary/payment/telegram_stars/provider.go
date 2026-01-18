package telegram_stars

import (
	"context"
	"fmt"

	"log/slog"

	"github.com/admin/tg-bots/astro-bot/internal/adapters/secondary/telegram"
	paymentPort "github.com/admin/tg-bots/astro-bot/internal/ports/payment"
)

// Provider реализует IPaymentProvider для Telegram Stars
// Поддерживает работу с несколькими ботами, каждый со своим Telegram client
type Provider struct {
	telegramClients map[string]*telegram.Client // botID → Client
	log             *slog.Logger
}

// NewProvider создаёт новый провайдер для Telegram Stars с поддержкой нескольких ботов
func NewProvider(telegramClients map[string]*telegram.Client, log *slog.Logger) *Provider {
	return &Provider{
		telegramClients: telegramClients,
		log:             log,
	}
}

// getClient получает Telegram client для указанного botID
func (p *Provider) getClient(botID string) (*telegram.Client, error) {
	client, ok := p.telegramClients[botID]
	if !ok {
		return nil, fmt.Errorf("telegram client not found for bot_id: %s", botID)
	}
	return client, nil
}

// CreateInvoice создаёт invoice для отправки пользователю через Telegram Stars
func (p *Provider) CreateInvoice(ctx context.Context, req paymentPort.CreateInvoiceRequest) (*paymentPort.CreateInvoiceResult, error) {
	// Получаем Telegram client для указанного бота
	client, err := p.getClient(req.BotID)
	if err != nil {
		return nil, fmt.Errorf("failed to get telegram client: %w", err)
	}

	// Формируем prices для invoice (для Stars это просто одна позиция)
	prices := []telegram.LabeledPrice{
		{
			Label:  req.ProductTitle,
			Amount: req.Amount, // количество звёзд
		},
	}

	// Создаём invoice через Telegram API
	invoiceReq := telegram.SendInvoiceRequest{
		ChatID:                    req.ChatID,
		Title:                     req.ProductTitle,
		Description:               req.Description,
		Payload:                   req.Payload,
		Currency:                  req.Currency, // "XTR" для Stars
		Prices:                    prices,
		PhotoURL:                  req.PhotoURL,
		PhotoSize:                 req.PhotoSize,
		PhotoWidth:                req.PhotoWidth,
		PhotoHeight:               req.PhotoHeight,
		NeedName:                  req.NeedName,
		NeedPhoneNumber:           req.NeedPhoneNumber,
		NeedEmail:                 req.NeedEmail,
		NeedShippingAddress:       req.NeedShippingAddress,
		SendPhoneNumberToProvider: req.SendPhoneNumberToProvider,
		SendEmailToProvider:       req.SendEmailToProvider,
		IsFlexible:                req.IsFlexible,
	}

	messageID, err := client.SendInvoice(ctx, invoiceReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send invoice: %w", err)
	}

	// Для Telegram Stars invoice_id = message_id (после отправки invoice)
	return &paymentPort.CreateInvoiceResult{
		InvoiceID: fmt.Sprintf("%d", messageID),
	}, nil
}

// ConfirmPreCheckout подтверждает pre_checkout_query (для Telegram Stars)
func (p *Provider) ConfirmPreCheckout(ctx context.Context, botID string, queryID string, ok bool, errorMessage *string) error {
	// Получаем Telegram client для указанного бота
	client, err := p.getClient(botID)
	if err != nil {
		return fmt.Errorf("failed to get telegram client: %w", err)
	}

	if err := client.AnswerPreCheckoutQuery(ctx, queryID, ok, errorMessage); err != nil {
		return fmt.Errorf("failed to answer pre_checkout_query: %w", err)
	}

	return nil
}
