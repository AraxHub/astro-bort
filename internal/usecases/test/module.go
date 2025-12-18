package testService

import (
	"context"
	"log/slog"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	ports "github.com/admin/tg-bots/astro-bot/internal/ports/repository"
)

type Service struct {
	TestRepo ports.ITestRepo
	Log      *slog.Logger
}

func New(testRepo ports.ITestRepo, log *slog.Logger) *Service {
	return &Service{
		TestRepo: testRepo,
		Log:      log,
	}
}

func (s *Service) SaveTest(ctx context.Context, test *domain.Test) error {
	if err := s.TestRepo.Create(ctx, test); err != nil {
		return err
	}

	return nil
}
