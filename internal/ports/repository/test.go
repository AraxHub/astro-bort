package repository

import (
	"context"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/admin/tg-bots/astro-bot/internal/ports/persistence"
)

type ITestRepo interface {
	Create(ctx context.Context, test *domain.Test) error
	GetByID(ctx context.Context, id int64) (*domain.Test, error)
	GetAll(ctx context.Context) ([]*domain.Test, error)
	Update(ctx context.Context, test *domain.Test) error
	DeleteById(ctx context.Context, id int64) error

	// Транзакции
	BeginTx(ctx context.Context) (persistence.Transaction, error)
	WithTransaction(ctx context.Context, fn func(context.Context, persistence.Transaction) error) error

	// Методы для работы в транзакции
	CreateTx(ctx context.Context, tx persistence.Transaction, test *domain.Test) error
	UpdateTx(ctx context.Context, tx persistence.Transaction, test *domain.Test) error
	DeleteTx(ctx context.Context, tx persistence.Transaction, id int64) error
	GetByIDTx(ctx context.Context, tx persistence.Transaction, id int64) (*domain.Test, error)
}
