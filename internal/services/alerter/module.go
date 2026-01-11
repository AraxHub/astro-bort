package alerter

import (
	"context"
	"fmt"

	"github.com/admin/tg-bots/astro-bot/internal/adapters/secondary/alerter"
	"github.com/admin/tg-bots/astro-bot/internal/ports/service"
)

// Service реализует IAlerterService для отправки алертов
type Service struct {
	client *alerter.Client
}

// New создаёт новый сервис для отправки алертов
func New(client *alerter.Client) service.IAlerterService {
	return &Service{
		client: client,
	}
}

// SendAlert отправляет алерт
func (s *Service) SendAlert(ctx context.Context, message string) error {
	if s.client == nil {
		return fmt.Errorf("alerter client is not initialized")
	}

	return s.client.SendAlert(ctx, message)
}
