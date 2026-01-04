package service

import (
	"context"
	"time"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
)

// IAstroAPIService интерфейс для работы с астро-API
type IAstroAPIService interface {
	CalculateNatalChart(ctx context.Context, birthDateTime time.Time, birthPlace string) (domain.NatalChart, error)
	GetNatalReport(ctx context.Context, birthDateTime time.Time, birthPlace string) (domain.NatalReport, error)
}
