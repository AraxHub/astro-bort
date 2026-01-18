package jobs

import (
	"context"
	"log/slog"
	"time"

	astroUsecase "github.com/admin/tg-bots/astro-bot/internal/usecases/astro"
)

const situationalWarningPushName = "situational-warning-push"

// SituationalWarningPush джоба для отправки пуша "ситуативное предупреждение"
// Отправляется в Ср 13:00 и Вс 9:00 по Мск
type SituationalWarningPush struct {
	astroService *astroUsecase.Service
	log          *slog.Logger
	location     *time.Location
}

// NewSituationalWarningPush создаёт новую джобу для отправки пуша "ситуативное предупреждение"
func NewSituationalWarningPush(
	astroService *astroUsecase.Service,
	log *slog.Logger,
) *SituationalWarningPush {
	location, _ := time.LoadLocation("Europe/Moscow")
	if location == nil {
		location = time.UTC
	}

	return &SituationalWarningPush{
		astroService: astroService,
		log:          log,
		location:     location,
	}
}

func (j *SituationalWarningPush) Name() string {
	return situationalWarningPushName
}

// NextRun вычисляет следующее время запуска (Ср 13:00 или Вс 9:00 по Мск)
func (j *SituationalWarningPush) NextRun(now time.Time) time.Time {
	nowMoscow := now.In(j.location)
	weekday := nowMoscow.Weekday()
	hour := nowMoscow.Hour()

	var nextDay time.Weekday
	var nextHour int

	// Определяем следующий день отправки
	if weekday <= time.Wednesday {
		// Если сейчас до среды включительно
		if weekday == time.Wednesday && hour >= 13 {
			// Уже прошло сегодня в среду, следующий - воскресенье
			nextDay = time.Sunday
			nextHour = 9
		} else {
			// Следующий - среда
			nextDay = time.Wednesday
			nextHour = 13
		}
	} else {
		// Если сейчас после среды (четверг-суббота)
		if weekday == time.Sunday && hour >= 9 {
			// Уже прошло сегодня в воскресенье, следующий - среда
			nextDay = time.Wednesday
			nextHour = 13
		} else {
			// Следующий - воскресенье
			nextDay = time.Sunday
			nextHour = 9
		}
	}

	// Вычисляем дни до следующего дня отправки
	daysUntil := (int(nextDay) - int(weekday) + 7) % 7
	if daysUntil == 0 && hour >= nextHour {
		// Уже прошло сегодня, следующий запуск через неделю
		if nextDay == time.Wednesday {
			daysUntil = 4 // до воскресенья (среда -> воскресенье)
		} else {
			daysUntil = 3 // до среды (воскресенье -> среда)
		}
	}

	next := nowMoscow.AddDate(0, 0, daysUntil)
	next = time.Date(next.Year(), next.Month(), next.Day(), nextHour, 0, 0, 0, j.location)

	return next
}

// Run отправляет пуш "ситуативное предупреждение" всем пользователям
func (j *SituationalWarningPush) Run(ctx context.Context) error {
	return j.astroService.SendSituationalWarningPush(ctx)
}
