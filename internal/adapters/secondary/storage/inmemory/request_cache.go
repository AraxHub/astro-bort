package inmemory

import (
	"sync"

	"github.com/admin/tg-bots/astro-bot/internal/ports/cache"
	"github.com/google/uuid"
)

// RequestCache in-memory реализация кэша последних request_id
type RequestCache struct {
	mu            sync.RWMutex
	lastRequestID map[int64]uuid.UUID          // chat_id -> request_id (для обратной совместимости)
	requestInfo   map[int64]*cache.RequestInfo // chat_id -> RequestInfo
}

// NewRequestCache создаёт новый in-memory кэш для request_id
func NewRequestCache() cache.IRequestCache {
	return &RequestCache{
		lastRequestID: make(map[int64]uuid.UUID),
		requestInfo:   make(map[int64]*cache.RequestInfo),
	}
}

// SetLastRequestID сохраняет последний request_id для chat_id
func (c *RequestCache) SetLastRequestID(chatID int64, requestID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastRequestID[chatID] = requestID
}

// IsLastRequestID проверяет, является ли request_id последним для данного chat_id
func (c *RequestCache) IsLastRequestID(chatID int64, requestID uuid.UUID) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	lastID, exists := c.lastRequestID[chatID]
	return exists && lastID == requestID
}

// SetRequestInfo сохраняет информацию о запросе (request_id и tech_msg_id)
func (c *RequestCache) SetRequestInfo(chatID int64, requestID uuid.UUID, techMsgID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastRequestID[chatID] = requestID
	c.requestInfo[chatID] = &cache.RequestInfo{
		RequestID:        requestID,
		TechMsgID:        techMsgID,
		IsTechMsgDeleted: false,
	}
}

// TryDeleteTechMsg атомарно проверяет и помечает техническое сообщение как удалённое
func (c *RequestCache) TryDeleteTechMsg(chatID int64, requestID uuid.UUID) (techMsgID int64, shouldDelete bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	info, exists := c.requestInfo[chatID]
	if !exists {
		return 0, false // нет информации о запросе
	}

	// Проверяем, что request_id совпадает (не устаревший запрос)
	if info.RequestID != requestID {
		return 0, false // устаревший запрос, не удаляем
	}

	// Проверяем, не удалено ли уже
	if info.IsTechMsgDeleted {
		return 0, false // уже удалено
	}

	// Помечаем как удалённое и возвращаем techMsgID
	info.IsTechMsgDeleted = true
	return info.TechMsgID, true
}
