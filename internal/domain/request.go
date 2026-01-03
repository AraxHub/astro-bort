package domain

import (
	"time"

	"github.com/google/uuid"
)

type Request struct {
	ID          uuid.UUID `json:"id" db:"id"`
	UserID      uuid.UUID `json:"user_id" db:"user_id"`
	BotID       BotId     `json:"bot_id" db:"bot_id"` // ID бота (для роутинга ответа)
	TGUpdateID  *int64    `json:"tg_update_id,omitempty" db:"tg_update_id"`
	RequestText string    `json:"request_text" db:"request_text"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}
