package statusRepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	ports "github.com/admin/tg-bots/astro-bot/internal/ports/repository"

	"log/slog"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/admin/tg-bots/astro-bot/internal/ports/persistence"
	"github.com/google/uuid"
)

type statusColumns struct {
	TableName    string
	ID           string
	ObjectType   string
	ObjectID     string
	Status       string
	ErrorMessage string
	Metadata     string
	CreatedAt    string
}

type Repository struct {
	db      persistence.Persistence
	Log     *slog.Logger
	columns statusColumns
}

// New создаёт новый репозиторий для работы со статусами
func New(db persistence.Persistence, log *slog.Logger) ports.IStatusRepo {
	cols := statusColumns{
		TableName:    "statuses",
		ID:           "id",
		ObjectType:   "object_type",
		ObjectID:     "object_id",
		Status:       "status",
		ErrorMessage: "error_message",
		Metadata:     "metadata",
		CreatedAt:    "created_at",
	}
	return &Repository{
		db:      db,
		Log:     log,
		columns: cols,
	}
}

// allColumns возвращает строку со всеми колонками
func (r *Repository) allColumns() string {
	return fmt.Sprintf("%s, %s, %s, %s, %s, %s, %s",
		r.columns.ID,
		r.columns.ObjectType,
		r.columns.ObjectID,
		r.columns.Status,
		r.columns.ErrorMessage,
		r.columns.Metadata,
		r.columns.CreatedAt)
}

// Create создает новый статус в таблице statuses
func (r *Repository) Create(ctx context.Context, status *domain.Status) error {
	query := fmt.Sprintf(`INSERT INTO %s (%s) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		r.columns.TableName,
		r.allColumns())
	err := r.db.Exec(ctx, query, status.ID, status.ObjectType, status.ObjectID, status.Status, status.ErrorMessage, status.Metadata, status.CreatedAt)
	if err != nil {
		r.Log.Error("failed to create status", "error", err, "object_type", status.ObjectType, "object_id", status.ObjectID, "status", status.Status)
		return fmt.Errorf("failed to create status: %w", err)
	}
	r.Log.Debug("status created successfully", "id", status.ID, "object_type", status.ObjectType, "object_id", status.ObjectID)
	return nil
}

// GetLatestByObjectID получает последний статус по типу объекта и его ID
func (r *Repository) GetLatestByObjectID(ctx context.Context, objectType domain.ObjectType, objectID uuid.UUID) (*domain.Status, error) {
	var status domain.Status
	query := fmt.Sprintf(`SELECT %s FROM %s WHERE object_type = $1 AND object_id = $2 ORDER BY created_at DESC LIMIT 1`,
		r.allColumns(),
		r.columns.TableName)
	err := r.db.Get(ctx, &status, query, objectType, objectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.Log.Warn("status not found", "object_type", objectType, "object_id", objectID)
			return nil, fmt.Errorf("status not found: %w", err)
		}
		r.Log.Error("failed to get latest status", "error", err, "object_type", objectType, "object_id", objectID)
		return nil, fmt.Errorf("failed to get latest status: %w", err)
	}
	r.Log.Debug("latest status retrieved successfully", "object_type", objectType, "object_id", objectID)
	return &status, nil
}

// GetByObjectID получает все статусы по типу объекта и его ID
func (r *Repository) GetByObjectID(ctx context.Context, objectType domain.ObjectType, objectID uuid.UUID) ([]*domain.Status, error) {
	var statuses []*domain.Status
	query := fmt.Sprintf(`SELECT %s FROM %s WHERE object_type = $1 AND object_id = $2 ORDER BY created_at ASC`,
		r.allColumns(),
		r.columns.TableName)
	err := r.db.Select(ctx, &statuses, query, objectType, objectID)
	if err != nil {
		r.Log.Error("failed to get statuses by object", "error", err, "object_type", objectType, "object_id", objectID)
		return nil, fmt.Errorf("failed to get statuses by object: %w", err)
	}
	r.Log.Debug("statuses retrieved successfully", "object_type", objectType, "object_id", objectID, "count", len(statuses))
	return statuses, nil
}

// GetByStatus получает все статусы по типу объекта и значению статуса
func (r *Repository) GetByStatus(ctx context.Context, objectType domain.ObjectType, status domain.RequestStatus) ([]*domain.Status, error) {
	var statuses []*domain.Status
	query := fmt.Sprintf(`SELECT %s FROM %s WHERE object_type = $1 AND status = $2 ORDER BY created_at ASC`,
		r.allColumns(),
		r.columns.TableName)
	err := r.db.Select(ctx, &statuses, query, objectType, status)
	if err != nil {
		r.Log.Error("failed to get statuses by status", "error", err, "object_type", objectType, "status", status)
		return nil, fmt.Errorf("failed to get statuses by status: %w", err)
	}
	r.Log.Debug("statuses retrieved successfully", "object_type", objectType, "status", status, "count", len(statuses))
	return statuses, nil
}

// BeginTx явно начинает транзакцию
func (r *Repository) BeginTx(ctx context.Context) (persistence.Transaction, error) {
	return r.db.BeginTx(ctx)
}

// WithTransaction выполняет функцию в транзакции с автоматическим commit/rollback
func (r *Repository) WithTransaction(ctx context.Context, fn func(context.Context, persistence.Transaction) error) error {
	return r.db.WithTransaction(ctx, fn)
}

// CreateTx создаёт статус в транзакции
func (r *Repository) CreateTx(ctx context.Context, tx persistence.Transaction, status *domain.Status) error {
	query := fmt.Sprintf(`INSERT INTO %s (%s) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		r.columns.TableName,
		r.allColumns())
	err := tx.Exec(ctx, query, status.ID, status.ObjectType, status.ObjectID, status.Status, status.ErrorMessage, status.Metadata, status.CreatedAt)
	if err != nil {
		r.Log.Error("failed to create status in transaction", "error", err, "object_type", status.ObjectType, "object_id", status.ObjectID)
		return fmt.Errorf("failed to create status in transaction: %w", err)
	}
	r.Log.Debug("status created in transaction", "id", status.ID, "object_type", status.ObjectType, "object_id", status.ObjectID)
	return nil
}

// GetLatestByObjectIDTx получает последний статус по типу объекта и его ID в транзакции
func (r *Repository) GetLatestByObjectIDTx(ctx context.Context, tx persistence.Transaction, objectType domain.ObjectType, objectID uuid.UUID) (*domain.Status, error) {
	var status domain.Status
	query := fmt.Sprintf(`SELECT %s FROM %s WHERE object_type = $1 AND object_id = $2 ORDER BY created_at DESC LIMIT 1`,
		r.allColumns(),
		r.columns.TableName)
	err := tx.Get(ctx, &status, query, objectType, objectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.Log.Warn("status not found in transaction", "object_type", objectType, "object_id", objectID)
			return nil, fmt.Errorf("status not found: %w", err)
		}
		r.Log.Error("failed to get latest status in transaction", "error", err, "object_type", objectType, "object_id", objectID)
		return nil, fmt.Errorf("failed to get latest status in transaction: %w", err)
	}
	r.Log.Debug("latest status retrieved in transaction", "object_type", objectType, "object_id", objectID)
	return &status, nil
}
