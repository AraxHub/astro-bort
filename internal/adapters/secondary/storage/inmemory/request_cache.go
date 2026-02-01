package inmemory

import (
	"sync"

	"github.com/google/uuid"
	"github.com/admin/tg-bots/astro-bot/internal/ports/cache"
)

// RequestCache in-memory реализация кэша последних request_id
type RequestCache struct {
	mu            sync.RWMutex
	lastRequestID map[int64]uuid.UUID // chat_id -> request_id
}

// NewRequestCache создаёт новый in-memory кэш для request_id
func NewRequestCache() cache.IRequestCache {
	return &RequestCache{
		lastRequestID: make(map[int64]uuid.UUID),
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
