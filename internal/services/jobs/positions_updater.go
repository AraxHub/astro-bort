package jobs

import (
	"context"
	"log/slog"
	"time"

	astroUsecase "github.com/admin/tg-bots/astro-bot/internal/usecases/astro"
)

const name = "positions-updater"

// PositionsUpdater джоба для обновления позиций планет в кеше Redis, каждый день в 05:00 по Мск
type PositionsUpdater struct {
	astroService *astroUsecase.Service
	log          *slog.Logger
	location     *time.Location
}

// NewPositionsUpdater создаёт новую джобу для обновления позиций планет
func NewPositionsUpdater(astroService *astroUsecase.Service, log *slog.Logger) *PositionsUpdater {
	location, _ := time.LoadLocation("Europe/Moscow")
	if location == nil {
		location = time.UTC
	}

	return &PositionsUpdater{
		astroService: astroService,
		log:          log,
		location:     location,
	}
}

func (j *PositionsUpdater) Name() string {
	return name
}

// NextRun вычисляет следующее время запуска
func (j *PositionsUpdater) NextRun(now time.Time) time.Time {
	nowMoscow := now.In(j.location)

	next := time.Date(nowMoscow.Year(), nowMoscow.Month(), nowMoscow.Day(), 5, 0, 0, 0, j.location)
	if next.Before(nowMoscow) || next.Equal(nowMoscow) {
		next = next.Add(24 * time.Hour)
	}
	return next
}

// Run выполняет обновление текущих позиций планет в кеше
func (j *PositionsUpdater) Run(ctx context.Context) error {
	now := time.Now().In(j.location)

	if err := j.astroService.UpdateCachedPositions(ctx, now); err != nil {
		return err
	}

	return nil
}
