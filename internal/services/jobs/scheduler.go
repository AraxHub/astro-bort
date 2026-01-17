package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/admin/tg-bots/astro-bot/internal/ports/jobs"
	"github.com/admin/tg-bots/astro-bot/internal/ports/service"
)

// Scheduler управляет запуском периодических джоб
type Scheduler struct {
	jobs           []jobs.Job
	alerterService service.IAlerterService
	log            *slog.Logger
}

// NewScheduler создаёт новый планировщик джоб
func NewScheduler(log *slog.Logger, alerterService service.IAlerterService) *Scheduler {
	return &Scheduler{
		jobs:           make([]jobs.Job, 0),
		alerterService: alerterService,
		log:            log,
	}
}

// Register регистрирует джобу в планировщике
func (s *Scheduler) Register(job jobs.Job) {
	s.jobs = append(s.jobs, job)
	s.log.Debug("job registered", "job_name", job.Name(), "total_jobs", len(s.jobs))
}

// Start запускает планировщик и все зарегистрированные джобы
func (s *Scheduler) Start(ctx context.Context) error {
	if len(s.jobs) == 0 {
		s.log.Error("no jobs registered, scheduler not started")
		return nil
	}

	s.log.Info("starting job scheduler", "jobs_count", len(s.jobs))

	for _, job := range s.jobs {
		jobName := job.Name()
		go func() {
			if err := s.runJob(ctx, job, jobName); err != nil {
				s.log.Error("job exited with error",
					"job_name", jobName,
					"error", err,
				)
			}
		}()
	}

	return nil
}

// runJob запускает отдельную джобу в цикле
func (s *Scheduler) runJob(ctx context.Context, job jobs.Job, jobName string) error {
	for {
		now := time.Now()
		nextRun := job.NextRun(now)

		duration := nextRun.Sub(now)

		select {
		case <-ctx.Done():
			s.log.Info("job stopped by context", "job_name", jobName)
			return nil
		case <-time.After(duration):
			err, errors := s.executeJobWithRetry(ctx, job, jobName)
			if err != nil {
				s.log.Error("job failed after all retries",
					"job_name", jobName,
					"error", err,
					"err 1", errors[0].error,
					"err 2", errors[1].error,
					"err 3", errors[2].error,
					"err 4", errors[3].error,
				)
				s.sendAlert(ctx, jobName, errors)
			} else {
				s.log.Info("job executed successfully", "job_name", jobName)
			}
		}
	}
}

// jobAttemptError представляет ошибку конкретной попытки выполнения джобы
type jobAttemptError struct {
	attempt int
	error   error
}

// executeJobWithRetry выполняет джобу с retry при ошибках | now + 1m + 10m + 30m
// Возвращает финальную ошибку и список всех ошибок попыток
func (s *Scheduler) executeJobWithRetry(ctx context.Context, job jobs.Job, jobName string) (error, []jobAttemptError) {
	retries := []time.Duration{
		1 * time.Minute,
		10 * time.Minute,
		30 * time.Minute,
	}

	var attemptErrors []jobAttemptError

	// Первая попытка
	if err := job.Run(ctx); err != nil {
		attemptErrors = append(attemptErrors, jobAttemptError{attempt: 1, error: err})
		s.log.Warn("job execution failed, will retry",
			"job_name", jobName,
			"attempt", 1,
			"retries_remaining", len(retries),
			"error", err,
		)
	} else {
		return nil, nil // Успех
	}

	// Retry
	for i, retryDelay := range retries {
		attemptNum := i + 2
		select {
		case <-ctx.Done():
			return ctx.Err(), attemptErrors
		case <-time.After(retryDelay):
			if err := job.Run(ctx); err != nil {
				attemptErrors = append(attemptErrors, jobAttemptError{attempt: attemptNum, error: err})
				s.log.Warn("job retry failed",
					"job_name", jobName,
					"attempt", attemptNum,
					"retries_remaining", len(retries)-i-1,
					"error", err,
				)
			} else {
				return nil, nil // Успех
			}
		}
	}

	finalErr := fmt.Errorf("all retry attempts failed (total attempts: %d)", 1+len(retries))
	return finalErr, attemptErrors
}

// sendAlert алертит на финальную ошибку после ретраев
func (s *Scheduler) sendAlert(ctx context.Context, jobName string, attemptErrors []jobAttemptError) {
	if s.alerterService == nil {
		return
	}

	var errorLines []string
	for _, attemptErr := range attemptErrors {
		errorLines = append(errorLines, fmt.Sprintf("Попытка %d: %s", attemptErr.attempt, attemptErr.error.Error()))
	}

	var message strings.Builder
	message.WriteString("⚠️ Финальная ошибка планировщика, ретраи исчерпаны\n@nhoj41_3\n\n")
	message.WriteString(fmt.Sprintf("Джоба: %s\n\n", jobName))
	message.WriteString("Ошибки попыток:\n")
	message.WriteString(strings.Join(errorLines, "\n"))

	if alertErr := s.alerterService.SendAlert(ctx, message.String()); alertErr != nil {
		s.log.Warn("failed to send job failure alert",
			"job_name", jobName,
			"error", alertErr,
		)
	}
}
