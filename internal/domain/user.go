package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID                      uuid.UUID  `json:"id" db:"id"`
	TelegramUserID          int64      `json:"telegram_user_id" db:"tg_id"`
	TelegramChatID          int64      `json:"telegram_chat_id" db:"chat_id"`
	FirstName               string     `json:"first_name" db:"first_name"`
	LastName                *string    `json:"last_name,omitempty" db:"last_name"`
	Username                *string    `json:"username,omitempty" db:"username"`
	BirthDateTime           *time.Time `json:"birth_datetime,omitempty" db:"birth_datetime"`
	BirthPlace              *string    `json:"birth_place,omitempty" db:"birth_place"`
	BirthDataSetAt          *time.Time `json:"birth_data_set_at,omitempty" db:"birth_data_set_at"`
	BirthDataCanChangeUntil *time.Time `json:"birth_data_can_change_until,omitempty" db:"birth_data_can_change_until"`
	NatalChart              NatalChart `json:"natal_chart,omitempty" db:"natal_chart"` // JSONB хранится как NatalChart
	NatalChartFetchedAt     *time.Time `json:"natal_chart_fetched_at,omitempty" db:"natal_chart_fetched_at"`
	CreatedAt               time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at" db:"updated_at"`
	LastSeenAt              *time.Time `json:"last_seen_at,omitempty" db:"last_seen_at"`
}
