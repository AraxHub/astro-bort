package repository

import (
	"context"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/admin/tg-bots/astro-bot/internal/ports/persistence"
	"github.com/google/uuid"
)

// IUserRepo интерфейс для работы с пользователями Telegram
type IUserRepo interface {
	Create(ctx context.Context, user *domain.User) error
	GetByTelegramID(ctx context.Context, telegramID int64) (*domain.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetNatalChart(ctx context.Context, userID uuid.UUID) (domain.NatalReport, error)
	Update(ctx context.Context, user *domain.User) error
	UpdateProfile(ctx context.Context, user *domain.User) error
	UpdateLastSeen(ctx context.Context, userID uuid.UUID) error
	UpdateFreeMsgCount(ctx context.Context, userID uuid.UUID) error
	SetPaidStatus(ctx context.Context, userID uuid.UUID, isPaid bool) error
	GetUsersWithExpiredSubscriptions(ctx context.Context) ([]uuid.UUID, error)
	RevokeExpiredSubscriptions(ctx context.Context) (int64, error)

	BeginTx(ctx context.Context) (persistence.Transaction, error)
	WithTransaction(ctx context.Context, fn func(context.Context, persistence.Transaction) error) error

	// Транзакционные методы
	CreateTx(ctx context.Context, tx persistence.Transaction, user *domain.User) error
	UpdateTx(ctx context.Context, tx persistence.Transaction, user *domain.User) error
	GetByTelegramIDTx(ctx context.Context, tx persistence.Transaction, telegramID int64) (*domain.User, error)
}
