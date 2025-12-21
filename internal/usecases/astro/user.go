package astro

import (
	"context"
	"fmt"
	"time"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/google/uuid"
)

// GetOrCreateUser получает пользователя по Telegram ID или создаёт нового
func (s *Service) GetOrCreateUser(ctx context.Context, tgUser *domain.TelegramUser, chat *domain.Chat) (*domain.User, error) {
	// Пытаемся найти существующего пользователя
	user, err := s.UserRepo.GetByTelegramID(ctx, tgUser.ID)
	if err == nil && user != nil {
		// Пользователь найден, обновляем данные если нужно
		needsUpdate := false
		if user.FirstName != tgUser.FirstName {
			user.FirstName = tgUser.FirstName
			needsUpdate = true
		}
		if (tgUser.LastName != nil && (user.LastName == nil || *user.LastName != *tgUser.LastName)) ||
			(tgUser.LastName == nil && user.LastName != nil) {
			user.LastName = tgUser.LastName
			needsUpdate = true
		}
		if (tgUser.Username != nil && (user.Username == nil || *user.Username != *tgUser.Username)) ||
			(tgUser.Username == nil && user.Username != nil) {
			user.Username = tgUser.Username
			needsUpdate = true
		}
		if user.TelegramChatID != chat.ID {
			user.TelegramChatID = chat.ID
			needsUpdate = true
		}

		if needsUpdate {
			user.UpdatedAt = time.Now()
			if err := s.UserRepo.Update(ctx, user); err != nil {
				s.Log.Warn("failed to update user",
					"error", err,
					"user_id", user.ID,
				)
			}
		}

		// Обновляем время последней активности
		if err := s.UserRepo.UpdateLastSeen(ctx, user.ID); err != nil {
			s.Log.Warn("failed to update last seen",
				"error", err,
				"user_id", user.ID,
			)
		}

		return user, nil
	}

	// Пользователь не найден, создаём нового
	now := time.Now()
	user = &domain.User{
		ID:             uuid.New(),
		TelegramUserID: tgUser.ID,
		TelegramChatID: chat.ID,
		FirstName:      tgUser.FirstName,
		LastName:       tgUser.LastName,
		Username:       tgUser.Username,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.UserRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	s.Log.Info("user created",
		"user_id", user.ID,
		"telegram_user_id", tgUser.ID,
	)

	return user, nil
}
