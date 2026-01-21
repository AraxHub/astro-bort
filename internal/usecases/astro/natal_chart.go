package astro

import (
	"context"
	"fmt"
	"time"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
)

// fetchAndSaveNatalChart получает натальную карту из астро-API и сохраняет её
func (s *Service) fetchAndSaveNatalChart(ctx context.Context, user *domain.User) error {
	if user.BirthDateTime == nil {
		return fmt.Errorf("birth date is not set")
	}

	if user.BirthPlace == nil || *user.BirthPlace == "" {
		return fmt.Errorf("birth place is not set")
	}

	natalReport, err := s.AstroAPIService.GetNatalReport(ctx, *user.BirthDateTime, *user.BirthPlace)
	if err != nil {
		return fmt.Errorf("failed to get natal report: %w", err)
	}

	if s.Cache != nil {
		cacheKey := fmt.Sprintf("astro:natal:%d", user.TelegramChatID)
		ttl := 24 * time.Hour
		if err := s.Cache.Set(ctx, cacheKey, string(natalReport), ttl); err != nil {
			s.Log.Warn("failed to cache natal chart in Redis",
				"error", err,
				"user_id", user.ID,
				"chat_id", user.TelegramChatID,
				"cache_key", cacheKey,
			)
		} else {
			s.Log.Debug("natal chart cached in Redis",
				"user_id", user.ID,
				"chat_id", user.TelegramChatID,
				"cache_key", cacheKey,
				"ttl", ttl,
			)
		}
	}

	now := time.Now()
	user.NatalChart = natalReport
	user.NatalChartFetchedAt = &now
	user.UpdatedAt = now

	if err := s.UserRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to save natal report: %w", err)
	}

	s.Log.Info("natal report saved",
		"user_id", user.ID,
		"birth_date", user.BirthDateTime.Format("02.01.2006"),
		"birth_place", *user.BirthPlace,
		"report_size", len(natalReport),
	)

	return nil
}
