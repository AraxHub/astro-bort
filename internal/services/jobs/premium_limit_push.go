package jobs

import (
	"context"
	"log/slog"
	"time"

	astroUsecase "github.com/admin/tg-bots/astro-bot/internal/usecases/astro"
)

const premiumLimitPushName = "premium-limit-push"

// PremiumLimitPush джоба для отправки пуша "платный лимит", каждая пятница в 13:00 по Мск
type PremiumLimitPush struct {
	astroService *astroUsecase.Service
	log          *slog.Logger
	location     *time.Location
}

// NewPremiumLimitPush создаёт новую джобу для отправки пуша "платный лимит"
func NewPremiumLimitPush(
	astroService *astroUsecase.Service,
	log *slog.Logger,
) *PremiumLimitPush {
	location, _ := time.LoadLocation("Europe/Moscow")
	if location == nil {
		location = time.UTC
	}

	return &PremiumLimitPush{
		astroService: astroService,
		log:          log,
		location:     location,
	}
}

func (j *PremiumLimitPush) Name() string {
	return premiumLimitPushName
}

// NextRun каждая пятница в 13:00 по Мск
func (j *PremiumLimitPush) NextRun(now time.Time) time.Time {
	nowMoscow := now.In(j.location)

	weekday := nowMoscow.Weekday()
	daysUntilFriday := (int(time.Friday) - int(weekday) + 7) % 7
	if daysUntilFriday == 0 && nowMoscow.Hour() >= 13 {
		daysUntilFriday = 7
	}

	next := nowMoscow.AddDate(0, 0, daysUntilFriday)
	next = time.Date(next.Year(), next.Month(), next.Day(), 13, 0, 0, 0, j.location)

	return next
}

func (j *PremiumLimitPush) Run(ctx context.Context) error {
	return j.astroService.SendPremiumLimitPush(ctx)
}
