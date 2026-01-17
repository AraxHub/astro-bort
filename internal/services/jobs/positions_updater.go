package jobs

import (
	"context"
	"log/slog"
	"time"

	jobPorts "github.com/admin/tg-bots/astro-bot/internal/ports/jobs"
	astroUsecase "github.com/admin/tg-bots/astro-bot/internal/usecases/astro"
)

// Проверяем, что PositionsUpdater реализует интерфейс jobs.Job
var _ jobPorts.Job = (*PositionsUpdater)(nil)

// PositionsUpdater джоба для обновления позиций планет в кеше Redis
// Запускается каждый день в 03:00 по Москве
type PositionsUpdater struct {
	astroService *astroUsecase.Service
	log          *slog.Logger
	location     *time.Location
}

// NewPositionsUpdater создаёт новую джобу для обновления позиций планет
func NewPositionsUpdater(astroService *astroUsecase.Service, log *slog.Logger) *PositionsUpdater {
	// Используем московское время для унификации
	location, _ := time.LoadLocation("Europe/Moscow")
	if location == nil {
		// Если не удалось загрузить, используем UTC
		location = time.UTC
	}

	return &PositionsUpdater{
		astroService: astroService,
		log:          log,
		location:     location,
	}
}

// NextRun вычисляет следующее время запуска (03:00 по Москве каждый день)
func (j *PositionsUpdater) NextRun(now time.Time) time.Time {
	nowMoscow := now.In(j.location)

	next := time.Date(nowMoscow.Year(), nowMoscow.Month(), nowMoscow.Day(), 10, 0, 0, 0, j.location)

	if next.Before(nowMoscow) || next.Equal(nowMoscow) {
		next = next.Add(24 * time.Hour)
	}

	return next
}

// NextRunTest вычисляет следующее время запуска для теста (через 10 секунд)
// ВРЕМЕННЫЙ МЕТОД ДЛЯ ТЕСТИРОВАНИЯ - удалить после теста
func (j *PositionsUpdater) NextRunTest(now time.Time) time.Time {
	return now.Add(10 * time.Second)
}

// Run выполняет обновление позиций планет в кеше
func (j *PositionsUpdater) Run(ctx context.Context) error {
	j.log.Info("updating cached planet positions")

	// Используем текущее время по Москве
	now := time.Now().In(j.location)

	if err := j.astroService.UpdateCachedPositions(ctx, now); err != nil {
		return err
	}

	j.log.Info("planet positions updated successfully")
	return nil
}
