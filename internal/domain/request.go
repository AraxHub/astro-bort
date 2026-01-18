package domain

import (
	"time"

	"github.com/google/uuid"
)

// RequestType тип запроса в системе
// Используется для различения обычных запросов пользователей и автоматических пушей
type RequestType string

const (
	// RequestTypeUser - обычный запрос от пользователя (через Telegram)
	// TGUpdateID != nil, RequestText содержит вопрос пользователя
	RequestTypeUser RequestType = "user"

	// RequestTypePushWeeklyForecast - автоматический пуш: прогноз на неделю
	// Отправляется в Пн 10:00, содержит универсальный прогноз на неделю
	RequestTypePushWeeklyForecast RequestType = "push_weekly_forecast"

	// RequestTypePushSituational - автоматический пуш: ситуативное предупреждение
	// Отправляется в Ср 13:00 и Вс 9:00, содержит прогноз на текущий день
	RequestTypePushSituational RequestType = "push_situational"

	// RequestTypePushPremiumLimit - автоматический пуш: напоминание о платном лимите
	// Отправляется в Пт 13:00, разный текст для бесплатников/платников
	RequestTypePushPremiumLimit RequestType = "push_premium_limit"
)

// IsPush проверяет, является ли запрос пушем
func (rt RequestType) IsPush() bool {
	return rt == RequestTypePushWeeklyForecast ||
		rt == RequestTypePushSituational ||
		rt == RequestTypePushPremiumLimit
}

// IsUser проверяет, является ли запрос обычным запросом от пользователя
func (rt RequestType) IsUser() bool {
	return rt == RequestTypeUser
}

// RequiresRAG проверяет, требует ли тип запроса отправки в RAG для генерации ответа
// Возвращает true для: user, push_weekly_forecast, push_situational
// Возвращает false для: push_premium_limit (хардкодный текст)
func (rt RequestType) RequiresRAG() bool {
	return rt == RequestTypeUser ||
		rt == RequestTypePushWeeklyForecast ||
		rt == RequestTypePushSituational
}

// IsHardcodedPush проверяет, является ли это пушом с заранее заданным текстом (без RAG)
func (rt RequestType) IsHardcodedPush() bool {
	return rt == RequestTypePushPremiumLimit
}

// IsRAGPush проверяет, является ли это пушем, который требует генерации через RAG
// (push_weekly_forecast, push_situational)
func (rt RequestType) IsRAGPush() bool {
	return rt == RequestTypePushWeeklyForecast ||
		rt == RequestTypePushSituational
}

type Request struct {
	ID           uuid.UUID   `json:"id" db:"id"`
	UserID       uuid.UUID   `json:"user_id" db:"user_id"`
	BotID        BotId       `json:"bot_id" db:"bot_id"`
	TGUpdateID   *int64      `json:"tg_update_id,omitempty" db:"tg_update_id"`
	RequestType  RequestType `json:"request_type" db:"request_type"`
	RequestText  string      `json:"request_text" db:"request_text"`
	ResponseText string      `json:"response_text" db:"response"`
	CreatedAt    time.Time   `json:"created_at" db:"created_at"`
}
