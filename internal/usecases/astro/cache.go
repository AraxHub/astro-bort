package astro

import (
	"context"
	"fmt"
	"time"
)

// UpdateCachedPositions обновляет актуальные позиции планет в кеше Redis.
// Использует фиксированный ключ astro:positions:current с TTL 25 часов.
// Предназначен для вызова из воркера, который обновляет данные раз в 24 часа.
// Возвращает ошибку, если не удалось запросить API или записать в кеш.
func (s *Service) UpdateCachedPositions(ctx context.Context, dateTime time.Time) error {
	const cacheKey = "astro:positions:current"

	if s.Cache == nil {
		s.Log.Warn("cache is not configured, skipping positions update")
		return nil
	}

	positions, err := s.AstroAPIService.GetPositions(ctx, dateTime)
	if err != nil {
		return fmt.Errorf("failed to get positions from API: %w", err)
	}

	ttl := 25 * time.Hour
	if err := s.Cache.Set(ctx, cacheKey, positions, ttl); err != nil {
		return fmt.Errorf("failed to cache positions: %w", err)
	}

	s.Log.Debug("positions cached successfully", "key", cacheKey, "ttl", ttl)
	return nil
}
