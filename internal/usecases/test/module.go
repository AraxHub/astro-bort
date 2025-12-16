package testService

import (
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

func (s *Service) SaveTest(test *domain.Test) error {
	if err := s.TestRepo.Create(test); err != nil {
		return err
	}

	return nil
}
