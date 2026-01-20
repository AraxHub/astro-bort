package astro

import (
	"context"
	"fmt"
	"time"
)

// UpdateCachedPositions обновляет актуальные позиции планет в кеше Redis.
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
	
	return nil
}
