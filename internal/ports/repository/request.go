package repository

import (
	"context"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/admin/tg-bots/astro-bot/internal/ports/persistence"
	"github.com/google/uuid"
)

// IRequestRepo интерфейс для работы с запросами пользователей
type IRequestRepo interface {
	Create(ctx context.Context, request *domain.Request) error
	UpdateResponseText(ctx context.Context, request *domain.Request) error
	UpdateResponseTextByID(ctx context.Context, requestID uuid.UUID, responseText string) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Request, error)
	GetByUpdateID(ctx context.Context, updateID int64) (*domain.Request, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Request, error)

	BeginTx(ctx context.Context) (persistence.Transaction, error)
	WithTransaction(ctx context.Context, fn func(context.Context, persistence.Transaction) error) error

	CreateTx(ctx context.Context, tx persistence.Transaction, request *domain.Request) error
	GetByIDTx(ctx context.Context, tx persistence.Transaction, id uuid.UUID) (*domain.Request, error)
	GetByUpdateIDTx(ctx context.Context, tx persistence.Transaction, updateID int64) (*domain.Request, error)
}
