package domain

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// PaymentMethod способ оплаты
type PaymentMethod string

const (
	PaymentMethodTelegramStars PaymentMethod = "telegram_stars"
)

// PaymentStatus статус платежа
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"   // создан, ожидает оплаты
	PaymentStatusSucceeded PaymentStatus = "succeeded" // успешно оплачен
	PaymentStatusFailed    PaymentStatus = "failed"    // оплата не прошла
	PaymentStatusRefunded  PaymentStatus = "refunded"  // возврат средств
)

// PaymentMetadata метаданные платежа (JSONB) с поддержкой sql.Scanner
type PaymentMetadata map[string]interface{}

// Scan реализует sql.Scanner для сканирования JSONB из БД
func (m *PaymentMetadata) Scan(value interface{}) error {
	if value == nil {
		*m = make(PaymentMetadata)
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		*m = make(PaymentMetadata)
		return nil
	}

	if len(bytes) == 0 {
		*m = make(PaymentMetadata)
		return nil
	}

	return json.Unmarshal(bytes, m)
}

// Value реализует driver.Valuer для сохранения в БД
func (m PaymentMetadata) Value() (driver.Value, error) {
	if len(m) == 0 {
		return "{}", nil
	}
	return json.Marshal(m)
}

// Payment платёж в системе
type Payment struct {
	ID           uuid.UUID       `json:"id" db:"id"`
	UserID       uuid.UUID       `json:"user_id" db:"user_id"`
	BotID        BotId           `json:"bot_id" db:"bot_id"`           // ID бота, в котором создан платёж
	Amount       int64           `json:"amount" db:"amount"`           // количество звёзд
	Currency     string          `json:"currency" db:"currency"`       // "XTR" для Stars
	Method       PaymentMethod   `json:"method" db:"method"`           // способ оплаты
	ProviderID   string          `json:"provider_id" db:"provider_id"` // ID в системе провайдера (Telegram invoice_id)
	Status       PaymentStatus   `json:"status" db:"status"`
	ProductID    string          `json:"product_id" db:"product_id"`       // что куплено (например, "premium_access")
	ProductTitle string          `json:"product_title" db:"product_title"` // название продукта для отображения
	Metadata     PaymentMetadata `json:"metadata,omitempty" db:"metadata"` // дополнительные данные (JSONB)
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
	SucceededAt  *time.Time      `json:"succeeded_at,omitempty" db:"succeeded_at"`
	FailedAt     *time.Time      `json:"failed_at,omitempty" db:"failed_at"`
	ErrorMessage *string         `json:"error_message,omitempty" db:"error_message"`
}
