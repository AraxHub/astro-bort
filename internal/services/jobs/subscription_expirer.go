package jobs

import (
	"context"
	"log/slog"
	"time"

	astroUsecase "github.com/admin/tg-bots/astro-bot/internal/usecases/astro"
)

const subscriptionExpirerName = "subscription-expirer"

// SubscriptionExpirer джоба для проверки и отзыва истёкших подписок, каждый день в 03:00 по Мск
type SubscriptionExpirer struct {
	astroService *astroUsecase.Service
	log          *slog.Logger
	location     *time.Location
}

func NewSubscriptionExpirer(
	astroService *astroUsecase.Service,
	log *slog.Logger,
) *SubscriptionExpirer {
	location, _ := time.LoadLocation("Europe/Moscow")
	if location == nil {
		location = time.UTC
	}

	return &SubscriptionExpirer{
		astroService: astroService,
		log:          log,
		location:     location,
	}
}

func (j *SubscriptionExpirer) Name() string {
	return subscriptionExpirerName
}

// NextRun каждый день в 03:00 по Мск
func (j *SubscriptionExpirer) NextRun(now time.Time) time.Time {
	nowMoscow := now.In(j.location)
	next := time.Date(nowMoscow.Year(), nowMoscow.Month(), nowMoscow.Day(), 3, 0, 0, 0, j.location)
	if next.Before(nowMoscow) || next.Equal(nowMoscow) {
		next = next.Add(24 * time.Hour)
	}
	return next
}

// Run проверяет и отзывает истёкшие подписки, отправляет уведомления пользователям
func (j *SubscriptionExpirer) Run(ctx context.Context) error {
	return j.astroService.RevokeExpiredSubscriptions(ctx)
}
