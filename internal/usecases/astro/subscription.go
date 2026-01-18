package astro

import (
	"context"
	"fmt"
	"time"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
)

// RevokeExpiredSubscriptions обрабатывает истёкшие подписки: отзывает их и уведомляет пользователей
// джоба для планировщика
func (s *Service) RevokeExpiredSubscriptions(ctx context.Context) error {
	expiredUserIDs, err := s.UserRepo.GetUsersWithExpiredSubscriptions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get users with expired subscriptions: %w", err)
	}

	if len(expiredUserIDs) == 0 {
		s.Log.Info("no expired subscriptions found")
		return nil
	}

	s.Log.Info("found expired subscriptions",
		"count", len(expiredUserIDs))

	rowsAffected, err := s.UserRepo.RevokeExpiredSubscriptions(ctx)
	if err != nil {
		return fmt.Errorf("failed to revoke expired subscriptions: %w", err)
	}

	s.Log.Info("expired subscriptions revoked",
		"count", rowsAffected)

	// Пауза 0.1 секунды = ~10 RPS, безопасно для лимита 30 RPS
	for i, userID := range expiredUserIDs {
		if i > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
			}
		}

		// Получаем пользователя для chat_id
		user, err := s.UserRepo.GetByID(ctx, userID)
		if err != nil {
			s.Log.Warn("failed to get user for notification",
				"error", err,
				"user_id", userID,
			)
			continue
		}

		// Получаем bot_id из последнего успешного платежа
		if s.PaymentRepo == nil {
			s.Log.Warn("payment repo not configured, cannot get bot_id",
				"user_id", userID,
			)
			continue
		}

		botID, err := s.PaymentRepo.GetBotIDForUser(ctx, userID)
		if err != nil {
			s.Log.Warn("failed to get bot_id for user",
				"error", err,
				"user_id", userID,
			)
			continue
		}

		message := fmt.Sprintf("🐱 Ой, я всё съела! За 30 дней подписка закончилась 🎉\n\n"+
			"🆓 У тебя осталось %d бесплатных сообщений 🐱", s.FreeMessagesLimit)

		if sendErr := s.sendMessage(ctx, domain.BotId(botID), user.TelegramChatID, message); sendErr != nil {
			s.Log.Warn("failed to send subscription expiry notification",
				"error", sendErr,
				"user_id", userID,
				"bot_id", botID,
			)
		}
	}

	return nil
}
