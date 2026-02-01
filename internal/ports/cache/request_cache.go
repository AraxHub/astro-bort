package cache

import "github.com/google/uuid"

// IRequestCache интерфейс для кэша последних request_id по chat_id
type IRequestCache interface {
	SetLastRequestID(chatID int64, requestID uuid.UUID)
	IsLastRequestID(chatID int64, requestID uuid.UUID) bool
}
