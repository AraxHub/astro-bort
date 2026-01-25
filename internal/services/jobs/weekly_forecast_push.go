package jobs

import (
	"context"
	"log/slog"
	"time"

	astroUsecase "github.com/admin/tg-bots/astro-bot/internal/usecases/astro"
)

const weeklyForecastPushName = "weekly-forecast-push"

// WeeklyForecastPush джоба для отправки пуша "прогноз на неделю", каждый понедельник в 10:00 по Мск
type WeeklyForecastPush struct {
	astroService *astroUsecase.Service
	log          *slog.Logger
	location     *time.Location
}

func NewWeeklyForecastPush(
	astroService *astroUsecase.Service,
	log *slog.Logger,
) *WeeklyForecastPush {
	location, _ := time.LoadLocation("Europe/Moscow")
	if location == nil {
		location = time.UTC
	}

	return &WeeklyForecastPush{
		astroService: astroService,
		log:          log,
		location:     location,
	}
}

func (j *WeeklyForecastPush) Name() string {
	return weeklyForecastPushName
}

// NextRun каждый понедельник в 10:00 по Мск
func (j *WeeklyForecastPush) NextRun(now time.Time) time.Time {
	nowMoscow := now.In(j.location)

	weekday := nowMoscow.Weekday()
	daysUntilMonday := (int(time.Monday) - int(weekday) + 7) % 7

	if daysUntilMonday == 0 && nowMoscow.Hour() >= 10 {
		daysUntilMonday = 7
	}

	next := nowMoscow.AddDate(0, 0, daysUntilMonday)
	next = time.Date(next.Year(), next.Month(), next.Day(), 10, 0, 0, 0, j.location)

	return next
}

func (j *WeeklyForecastPush) Run(ctx context.Context) error {
	return j.astroService.SendWeeklyForecastPush(ctx)
}
