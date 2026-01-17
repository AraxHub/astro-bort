package jobs

import (
	"context"
	"time"
)

// Job задача, которую можно запланировать
type Job interface {
	Name() string
	NextRun(now time.Time) time.Time
	Run(ctx context.Context) error
}
