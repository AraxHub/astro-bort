package cache

import "github.com/google/uuid"

// RequestInfo информация о запросе для удаления технического сообщения
type RequestInfo struct {
	RequestID        uuid.UUID
	TechMsgID        int64
	IsTechMsgDeleted bool
}

// IRequestCache интерфейс для кэша последних request_id по chat_id
type IRequestCache interface {
	SetLastRequestID(chatID int64, requestID uuid.UUID)
	IsLastRequestID(chatID int64, requestID uuid.UUID) bool
	SetRequestInfo(chatID int64, requestID uuid.UUID, techMsgID int64)
	TryDeleteTechMsg(chatID int64, requestID uuid.UUID) (techMsgID int64, shouldDelete bool)
}
