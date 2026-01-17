package jobs

import (
	"context"
	"time"
)

// Job представляет периодическую задачу, которую можно запланировать
type Job interface {
	NextRun(now time.Time) time.Time
	Run(ctx context.Context) error
}
