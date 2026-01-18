package domain

// дока - https://core.telegram.org/bots/api

// Update - входящее обновление от Telegram Bot API
type Update struct {
	UpdateID         int64             `json:"update_id"`
	Message          *Message          `json:"message,omitempty"`
	CallbackQuery    *CallbackQuery    `json:"callback_query,omitempty"`
	PreCheckoutQuery *PreCheckoutQuery `json:"pre_checkout_query,omitempty"` // для обработки платежей Stars
	// Можно добавить другие типы обновлений по мере необходимости:
	// EditedMessage      *Message `json:"edited_message,omitempty"`
	// и т.д.
}

// CallbackQuery - callback query от Telegram Bot API
type CallbackQuery struct {
	ID      string        `json:"id"`
	From    *TelegramUser `json:"from,omitempty"`
	Message *Message      `json:"message,omitempty"`
	Data    *string       `json:"data,omitempty"` // данные callback кнопки
}

// Message - сообщение от Telegram Bot API
type Message struct {
	MessageID         int64              `json:"message_id"`
	From              *TelegramUser      `json:"from,omitempty"`               // отправитель (Telegram User)
	Chat              *Chat              `json:"chat"`                         // чат
	Date              int64              `json:"date"`                         // Unix timestamp
	Text              *string            `json:"text,omitempty"`               // текст сообщения
	Entities          []Entity           `json:"entities,omitempty"`           // сущности (команды, упоминания и т.д.)
	SuccessfulPayment *SuccessfulPayment `json:"successful_payment,omitempty"` // успешный платёж Stars
}

// User - пользователя Telegram (не domain.User)
type TelegramUser struct {
	ID           int64   `json:"id"`
	IsBot        bool    `json:"is_bot"`
	FirstName    string  `json:"first_name"`
	LastName     *string `json:"last_name,omitempty"`
	Username     *string `json:"username,omitempty"`
	LanguageCode *string `json:"language_code,omitempty"`
}

// Chat - чат в Telegram
type Chat struct {
	ID        int64   `json:"id"`
	Type      string  `json:"type"` // "private", "group", "supergroup", "channel"
	Title     *string `json:"title,omitempty"`
	Username  *string `json:"username,omitempty"`
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
}

// PreCheckoutQuery - запрос на подтверждение платежа (Telegram Stars)
// Документация: https://core.telegram.org/bots/api#precheckoutquery
type PreCheckoutQuery struct {
	ID               string        `json:"id"`
	From             *TelegramUser `json:"from,omitempty"`
	Currency         string        `json:"currency"`        // "XTR" для Stars
	TotalAmount      int64         `json:"total_amount"`    // количество звёзд
	InvoicePayload   string        `json:"invoice_payload"` // payload, который мы передали при создании invoice
	ShippingOptionID *string       `json:"shipping_option_id,omitempty"`
	OrderInfo        *OrderInfo    `json:"order_info,omitempty"`
}

// SuccessfulPayment - успешный платёж (Telegram Stars)
// Документация: https://core.telegram.org/bots/api#successfulpayment
type SuccessfulPayment struct {
	Currency                string     `json:"currency"`                             // "XTR" для Stars
	TotalAmount             int64      `json:"total_amount"`                         // количество звёзд
	InvoicePayload          string     `json:"invoice_payload"`                      // payload, который мы передали при создании invoice
	TelegramPaymentChargeID string     `json:"telegram_payment_charge_id"`           // ID платежа в системе Telegram
	ProviderPaymentChargeID *string    `json:"provider_payment_charge_id,omitempty"` // для внешних провайдеров
	ShippingOptionID        *string    `json:"shipping_option_id,omitempty"`
	OrderInfo               *OrderInfo `json:"order_info,omitempty"`
}

// OrderInfo - информация о заказе (опционально)
type OrderInfo struct {
	Name            *string          `json:"name,omitempty"`
	PhoneNumber     *string          `json:"phone_number,omitempty"`
	Email           *string          `json:"email,omitempty"`
	ShippingAddress *ShippingAddress `json:"shipping_address,omitempty"`
}

// ShippingAddress - адрес доставки (опционально)
type ShippingAddress struct {
	CountryCode string  `json:"country_code"`
	State       *string `json:"state,omitempty"`
	City        string  `json:"city"`
	StreetLine1 string  `json:"street_line1"`
	StreetLine2 *string `json:"street_line2,omitempty"`
	PostCode    string  `json:"post_code"`
}

// Entity - сущность в сообщении (команда, упоминание и т.д.)
type Entity struct {
	Type   string `json:"type"`   // "bot_command", "mention", "url" и т.д.
	Offset int    `json:"offset"` // смещение в UTF-16 кодовых единицах
	Length int    `json:"length"` // длина в UTF-16 кодовых единицах
}

type BotType string

const (
	BotTypeAstro BotType = "astro"
)

func (bt BotType) String() string {
	return string(bt)
}

func (bt BotType) IsValid() bool {
	switch bt {
	case BotTypeAstro:
		return true
	default:
		return false
	}
}

type BotId string
