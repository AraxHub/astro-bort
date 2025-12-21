package repository

import (
	"context"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/admin/tg-bots/astro-bot/internal/ports/persistence"
	"github.com/google/uuid"
)

// IStatusRepo интерфейс для работы со статусами (event sourcing)
type IStatusRepo interface {
	Create(ctx context.Context, status *domain.Status) error
	GetLatestByObjectID(ctx context.Context, objectType domain.ObjectType, objectID uuid.UUID) (*domain.Status, error)
	GetByObjectID(ctx context.Context, objectType domain.ObjectType, objectID uuid.UUID) ([]*domain.Status, error)
	GetByStatus(ctx context.Context, objectType domain.ObjectType, status domain.RequestStatus) ([]*domain.Status, error)

	BeginTx(ctx context.Context) (persistence.Transaction, error)
	WithTransaction(ctx context.Context, fn func(context.Context, persistence.Transaction) error) error

	CreateTx(ctx context.Context, tx persistence.Transaction, status *domain.Status) error
	GetLatestByObjectIDTx(ctx context.Context, tx persistence.Transaction, objectType domain.ObjectType, objectID uuid.UUID) (*domain.Status, error)
}
