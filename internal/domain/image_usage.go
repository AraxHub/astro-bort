package domain

import (
	"time"
)

// ImageUsage представляет статистику использования картинок по чату
type ImageUsage struct {
	ChatID     int64            `json:"chat_id" db:"chat_id"`
	UsedImages map[string]int   `json:"used_images" db:"used_images"` // JSONB: {"L1": 3, "L2": 2}
	CreatedAt  time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at" db:"updated_at"`
}
