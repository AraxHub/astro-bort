package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/admin/tg-bots/astro-bot/internal/ports/jobs"
)

// Scheduler управляет запуском периодических джоб
type Scheduler struct {
	jobs []jobs.Job
	log  *slog.Logger
}

// NewScheduler создаёт новый планировщик джоб
func NewScheduler(log *slog.Logger) *Scheduler {
	return &Scheduler{
		jobs: make([]jobs.Job, 0),
		log:  log,
	}
}

// Register регистрирует джобу в планировщике
func (s *Scheduler) Register(job jobs.Job) {
	s.jobs = append(s.jobs, job)
	s.log.Debug("job registered", "total_jobs", len(s.jobs))
}

// Start запускает планировщик и все зарегистрированные джобы
func (s *Scheduler) Start(ctx context.Context) error {
	if len(s.jobs) == 0 {
		s.log.Info("no jobs registered, scheduler not started")
		return nil
	}

	s.log.Info("starting job scheduler", "jobs_count", len(s.jobs))

	for i, job := range s.jobs {
		go func() {
			if err := s.runJob(ctx, job, i); err != nil {
				s.log.Error("job exited with error",
					"job_index", i,
					"error", err,
				)
			}
		}()
	}

	return nil
}

// runJob запускает отдельную джобу в цикле
func (s *Scheduler) runJob(ctx context.Context, job jobs.Job, jobIndex int) error {
	for {
		now := time.Now()
		// ВРЕМЕННО для теста: используем NextRunTest вместо NextRun
		// TODO: вернуть job.NextRun(now) после тестирования
		var nextRun time.Time
		if testJob, ok := job.(interface{ NextRunTest(time.Time) time.Time }); ok {
			nextRun = testJob.NextRunTest(now)
		} else {
			nextRun = job.NextRun(now)
		}

		duration := nextRun.Sub(now)
		s.log.Debug("job scheduled",
			"job_index", jobIndex,
			"next_run", nextRun,
			"duration", duration,
		)

		// Ждём до следующего запуска или отмены контекста
		select {
		case <-ctx.Done():
			s.log.Info("job stopped by context", "job_index", jobIndex)
			return nil
		case <-time.After(duration):
			// Запускаем джобу
			if err := job.Run(ctx); err != nil {
				s.log.Error("job execution failed",
					"job_index", jobIndex,
					"error", err,
				)
				// Продолжаем выполнение даже при ошибке
			} else {
				s.log.Debug("job executed successfully", "job_index", jobIndex)
			}
		}
	}
}
