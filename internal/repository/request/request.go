package requestRepo

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

type requestColumns struct {
	TableName    string
	ID           string
	UserID       string
	BotID        string
	TGUpdateID   string
	RequestText  string
	ResponseText string
	CreatedAt    string
}

type Repository struct {
	db      persistence.Persistence
	Log     *slog.Logger
	columns requestColumns
}

// New создаёт новый репозиторий для работы с запросами
func New(db persistence.Persistence, log *slog.Logger) ports.IRequestRepo {
	cols := requestColumns{
		TableName:    "requests",
		ID:           "id",
		UserID:       "user_id",
		BotID:        "bot_id",
		TGUpdateID:   "tg_update_id",
		RequestText:  "request_text",
		ResponseText: "response",
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
		r.columns.UserID,
		r.columns.BotID,
		r.columns.TGUpdateID,
		r.columns.RequestText,
		r.columns.ResponseText,
		r.columns.CreatedAt)
}

// Create создаёт новый запрос
func (r *Repository) Create(ctx context.Context, request *domain.Request) error {
	query := fmt.Sprintf(`INSERT INTO %s (%s, %s, %s, %s, %s, %s) VALUES ($1, $2, $3, $4, $5, $6)`,
		r.columns.TableName,
		r.columns.ID,
		r.columns.UserID,
		r.columns.BotID,
		r.columns.TGUpdateID,
		r.columns.RequestText,
		r.columns.CreatedAt)
	err := r.db.Exec(ctx, query,
		request.ID,
		request.UserID,
		request.BotID,
		request.TGUpdateID,
		request.RequestText,
		request.CreatedAt)
	if err != nil {
		r.Log.Error("failed to create request",
			"error", err,
			"user_id", request.UserID,
			"request_id", request.ID)
		return fmt.Errorf("failed to create request: %w", err)
	}
	r.Log.Debug("request created successfully",
		"id", request.ID,
		"user_id", request.UserID)
	return nil
}

// UpdateResponseText обновляет ответ в запросе
func (r *Repository) UpdateResponseText(ctx context.Context, request *domain.Request) error {
	query := fmt.Sprintf(`UPDATE %s SET %s = $1 WHERE %s = $2`,
		r.columns.TableName,
		r.columns.ResponseText,
		r.columns.ID)
	err := r.db.Exec(ctx, query, request.ResponseText, request.ID)
	if err != nil {
		r.Log.Error("failed to update request",
			"error", err,
			"request_id", request.ID,
		)
		return fmt.Errorf("failed to update request: %w", err)
	}
	r.Log.Debug("request updated successfully",
		"request_id", request.ID,
	)
	return nil
}

// UpdateResponseTextByID обновляет ответ в запросе напрямую по ID (без SELECT)
func (r *Repository) UpdateResponseTextByID(ctx context.Context, requestID uuid.UUID, responseText string) error {
	query := fmt.Sprintf(`UPDATE %s SET %s = $1 WHERE %s = $2`,
		r.columns.TableName,
		r.columns.ResponseText,
		r.columns.ID)
	err := r.db.Exec(ctx, query, responseText, requestID)
	if err != nil {
		r.Log.Error("failed to update request by id",
			"error", err,
			"request_id", requestID,
		)
		return fmt.Errorf("failed to update request: %w", err)
	}
	r.Log.Debug("request updated successfully by id",
		"request_id", requestID,
	)
	return nil
}

// GetByID получает запрос по ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Request, error) {
	var request domain.Request
	query := fmt.Sprintf(`SELECT %s FROM %s WHERE %s = $1`,
		r.allColumns(),
		r.columns.TableName,
		r.columns.ID)
	err := r.db.Get(ctx, &request, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.Log.Warn("request not found", "request_id", id)
			return nil, fmt.Errorf("request not found: %w", err)
		}
		r.Log.Error("failed to get request by id",
			"error", err,
			"request_id", id)
		return nil, fmt.Errorf("failed to get request by id: %w", err)
	}
	r.Log.Debug("request retrieved successfully", "request_id", id)
	return &request, nil
}

// GetByUpdateID получает запрос по Telegram Update ID
func (r *Repository) GetByUpdateID(ctx context.Context, updateID int64) (*domain.Request, error) {
	var request domain.Request
	query := fmt.Sprintf(`SELECT %s FROM %s WHERE %s = $1`,
		r.allColumns(),
		r.columns.TableName,
		r.columns.TGUpdateID)
	err := r.db.Get(ctx, &request, query, updateID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.Log.Warn("request not found", "tg_update_id", updateID)
			return nil, fmt.Errorf("request not found: %w", err)
		}
		r.Log.Error("failed to get request by update id",
			"error", err,
			"tg_update_id", updateID)
		return nil, fmt.Errorf("failed to get request by update id: %w", err)
	}
	r.Log.Debug("request retrieved successfully", "tg_update_id", updateID, "request_id", request.ID)
	return &request, nil
}

// GetByUserID получает все запросы пользователя
func (r *Repository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.Request, error) {
	var requests []*domain.Request
	query := fmt.Sprintf(`SELECT %s FROM %s WHERE %s = $1 ORDER BY %s DESC`,
		r.allColumns(),
		r.columns.TableName,
		r.columns.UserID,
		r.columns.CreatedAt)
	err := r.db.Select(ctx, &requests, query, userID)
	if err != nil {
		r.Log.Error("failed to get requests by user id",
			"error", err,
			"user_id", userID)
		return nil, fmt.Errorf("failed to get requests by user id: %w", err)
	}
	r.Log.Debug("requests retrieved successfully",
		"user_id", userID,
		"count", len(requests))
	return requests, nil
}

// BeginTx явно начинает транзакцию
func (r *Repository) BeginTx(ctx context.Context) (persistence.Transaction, error) {
	return r.db.BeginTx(ctx)
}

// WithTransaction выполняет функцию в транзакции с автоматическим commit/rollback
func (r *Repository) WithTransaction(ctx context.Context, fn func(context.Context, persistence.Transaction) error) error {
	return r.db.WithTransaction(ctx, fn)
}

// CreateTx создаёт запрос в транзакции
func (r *Repository) CreateTx(ctx context.Context, tx persistence.Transaction, request *domain.Request) error {
	query := fmt.Sprintf(`INSERT INTO %s (%s, %s, %s, %s, %s, %s) VALUES ($1, $2, $3, $4, $5, $6)`,
		r.columns.TableName,
		r.columns.ID,
		r.columns.UserID,
		r.columns.BotID,
		r.columns.TGUpdateID,
		r.columns.RequestText,
		r.columns.CreatedAt)
	err := tx.Exec(ctx, query,
		request.ID,
		request.UserID,
		request.BotID,
		request.TGUpdateID,
		request.RequestText,
		request.CreatedAt)
	if err != nil {
		r.Log.Error("failed to create request in transaction",
			"error", err,
			"user_id", request.UserID,
			"request_id", request.ID)
		return fmt.Errorf("failed to create request in transaction: %w", err)
	}
	r.Log.Debug("request created in transaction",
		"id", request.ID,
		"user_id", request.UserID)
	return nil
}

// GetByIDTx получает запрос по ID в транзакции
func (r *Repository) GetByIDTx(ctx context.Context, tx persistence.Transaction, id uuid.UUID) (*domain.Request, error) {
	var request domain.Request
	query := fmt.Sprintf(`SELECT %s FROM %s WHERE %s = $1`,
		r.allColumns(),
		r.columns.TableName,
		r.columns.ID)
	err := tx.Get(ctx, &request, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.Log.Warn("request not found in transaction", "request_id", id)
			return nil, fmt.Errorf("request not found: %w", err)
		}
		r.Log.Error("failed to get request by id in transaction",
			"error", err,
			"request_id", id)
		return nil, fmt.Errorf("failed to get request by id in transaction: %w", err)
	}
	r.Log.Debug("request retrieved in transaction", "request_id", id)
	return &request, nil
}

// GetByUpdateIDTx получает запрос по Telegram Update ID в транзакции
func (r *Repository) GetByUpdateIDTx(ctx context.Context, tx persistence.Transaction, updateID int64) (*domain.Request, error) {
	var request domain.Request
	query := fmt.Sprintf(`SELECT %s FROM %s WHERE %s = $1`,
		r.allColumns(),
		r.columns.TableName,
		r.columns.TGUpdateID)
	err := tx.Get(ctx, &request, query, updateID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.Log.Warn("request not found in transaction", "tg_update_id", updateID)
			return nil, fmt.Errorf("request not found: %w", err)
		}
		r.Log.Error("failed to get request by update id in transaction",
			"error", err,
			"tg_update_id", updateID)
		return nil, fmt.Errorf("failed to get request by update id in transaction: %w", err)
	}
	r.Log.Debug("request retrieved in transaction", "tg_update_id", updateID, "request_id", request.ID)
	return &request, nil
}
