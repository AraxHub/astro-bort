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

	natalChart, err := s.AstroAPIService.CalculateNatalChart(ctx, *user.BirthDateTime, *user.BirthPlace)
	if err != nil {
		return fmt.Errorf("failed to calculate natal chart: %w", err)
	}

	now := time.Now()
	user.NatalChart = natalChart
	user.NatalChartFetchedAt = &now
	user.UpdatedAt = now

	if err := s.UserRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to save natal chart: %w", err)
	}

	s.Log.Info("natal chart saved",
		"user_id", user.ID,
		"birth_date", user.BirthDateTime.Format("02.01.2006"),
		"birth_place", *user.BirthPlace,
		"chart_size", len(natalChart),
	)

	return nil
}
