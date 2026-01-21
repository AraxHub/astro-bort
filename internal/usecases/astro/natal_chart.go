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
		// Алертим об ошибке получения натальной карты
		if s.AlerterService != nil {
			alertMsg := fmt.Sprintf("❌ Отъебнула астро-апи\n\n@nhoj41_3 @matarseks @romanovnl\n\nFailed to get natal report\nUser ID: `%s`\nChat ID: `%d`\nBirth Date: %s\nBirth Place: %s\nError: %s",
				user.ID, user.TelegramChatID,
				user.BirthDateTime.Format("02.01.2006 15:04"),
				*user.BirthPlace,
				err.Error())
			if alertErr := s.AlerterService.SendAlert(ctx, alertMsg); alertErr != nil {
				s.Log.Warn("failed to send alert for natal chart error", "error", alertErr)
			}
		}
		return fmt.Errorf("failed to get natal report: %w", err)
	}

	// Проверяем, что ответ не пустой (на случай если API вернул 200 но пустой body)
	if len(natalReport) == 0 {
		err := fmt.Errorf("astro API returned empty response")
		if s.AlerterService != nil {
			alertMsg := fmt.Sprintf("❌ Отъебнула астро-апи\n\n@nhoj41_3 @matarseks @romanovnl\n\nFailed to get natal report\nUser ID: `%s`\nChat ID: `%d`\nBirth Date: %s\nBirth Place: %s",
				user.ID, user.TelegramChatID,
				user.BirthDateTime.Format("02.01.2006 15:04"),
				*user.BirthPlace)
			if alertErr := s.AlerterService.SendAlert(ctx, alertMsg); alertErr != nil {
				s.Log.Warn("failed to send alert for empty response", "error", alertErr)
			}
		}
		return err
	}

	// Сохраняем в Redis только если ответ валидный
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

	// Сохраняем в БД только если ответ валидный
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
