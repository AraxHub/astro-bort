package testRepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	ports "github.com/admin/tg-bots/astro-bot/internal/ports/repository"

	"log/slog"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/admin/tg-bots/astro-bot/internal/ports/persistence"
)

type Repository struct {
	db  persistence.Persistence
	Log *slog.Logger
}

// New создаёт новый репозиторий для работы с тестовой таблицей
func New(db persistence.Persistence, log *slog.Logger) ports.ITestRepo {
	return &Repository{
		db:  db,
		Log: log,
	}
}

// Create создает новую запись в таблице test
func (r *Repository) Create(ctx context.Context, test *domain.Test) error {
	query := `INSERT INTO test (filed1, filed2) VALUES ($1, $2) RETURNING id`
	err := r.db.QueryRow(ctx, query, test.Filed1, test.Filed2).Scan(&test.ID)
	if err != nil {
		r.Log.Error("failed to create test", "error", err, "filed1", test.Filed1, "filed2", test.Filed2)
		return fmt.Errorf("failed to create test: %w", err)
	}
	r.Log.Debug("test created successfully", "id", test.ID)
	return nil
}

// GetByID получает запись по ID
func (r *Repository) GetByID(ctx context.Context, id int64) (*domain.Test, error) {
	var test domain.Test
	query := `SELECT id, filed1, filed2 FROM test WHERE id = $1`
	err := r.db.Get(ctx, &test, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.Log.Warn("test not found", "id", id)
			return nil, fmt.Errorf("test not found: %w", err)
		}
		r.Log.Error("failed to get test", "error", err, "id", id)
		return nil, fmt.Errorf("failed to get test: %w", err)
	}
	r.Log.Debug("test retrieved successfully", "id", id)
	return &test, nil
}

// GetAll получает все записи из таблицы test
func (r *Repository) GetAll(ctx context.Context) ([]*domain.Test, error) {
	var tests []*domain.Test
	query := `SELECT id, filed1, filed2 FROM test ORDER BY id`
	err := r.db.Select(ctx, &tests, query)
	if err != nil {
		r.Log.Error("failed to get all tests", "error", err)
		return nil, fmt.Errorf("failed to get all tests: %w", err)
	}
	r.Log.Debug("tests retrieved successfully", "count", len(tests))
	return tests, nil
}

// Update обновляет запись в таблице test
func (r *Repository) Update(ctx context.Context, test *domain.Test) error {
	query := `UPDATE test SET filed1 = :filed1, filed2 = :filed2 WHERE id = :id`
	rowsAffected, err := r.db.NamedExecWithResult(ctx, query, test)
	if err != nil {
		r.Log.Error("failed to update test", "error", err, "id", test.ID)
		return fmt.Errorf("failed to update test: %w", err)
	}

	if rowsAffected == 0 {
		r.Log.Warn("test not found for update", "id", test.ID)
		return fmt.Errorf("test not found")
	}

	r.Log.Debug("test updated successfully", "id", test.ID, "rowsAffected", rowsAffected)
	return nil
}

// DeleteById удаляет запись по ID
func (r *Repository) DeleteById(ctx context.Context, id int64) error {
	query := `DELETE FROM test WHERE id = $1`
	rowsAffected, err := r.db.ExecWithResult(ctx, query, id)
	if err != nil {
		r.Log.Error("failed to delete test", "error", err, "id", id)
		return fmt.Errorf("failed to delete test: %w", err)
	}

	if rowsAffected == 0 {
		r.Log.Warn("test not found for delete", "id", id)
		return fmt.Errorf("test not found")
	}

	r.Log.Debug("test deleted successfully", "id", id, "rowsAffected", rowsAffected)
	return nil
}

// BeginTx явно начинает транзакцию
func (r *Repository) BeginTx(ctx context.Context) (persistence.Transaction, error) {
	return r.db.BeginTx(ctx)
}

// WithTransaction выполняет функцию в транзакции с автоматическим commit/rollback
func (r *Repository) WithTransaction(ctx context.Context, fn func(context.Context, persistence.Transaction) error) error {
	return r.db.WithTransaction(ctx, fn)
}

// CreateTx создаёт запись в транзакции
func (r *Repository) CreateTx(ctx context.Context, tx persistence.Transaction, test *domain.Test) error {
	query := `INSERT INTO test (filed1, filed2) VALUES ($1, $2) RETURNING id`
	err := tx.QueryRow(ctx, query, test.Filed1, test.Filed2).Scan(&test.ID)
	if err != nil {
		r.Log.Error("failed to create test in transaction", "error", err, "filed1", test.Filed1)
		return fmt.Errorf("failed to create test in transaction: %w", err)
	}
	r.Log.Debug("test created in transaction", "id", test.ID)
	return nil
}

// UpdateTx обновляет запись в транзакции
func (r *Repository) UpdateTx(ctx context.Context, tx persistence.Transaction, test *domain.Test) error {
	query := `UPDATE test SET filed1 = :filed1, filed2 = :filed2 WHERE id = :id`
	rowsAffected, err := tx.NamedExecWithResult(ctx, query, test)
	if err != nil {
		r.Log.Error("failed to update test in transaction", "error", err, "id", test.ID)
		return fmt.Errorf("failed to update test in transaction: %w", err)
	}

	if rowsAffected == 0 {
		r.Log.Warn("test not found for update in transaction", "id", test.ID)
		return fmt.Errorf("test not found")
	}

	r.Log.Debug("test updated in transaction", "id", test.ID, "rowsAffected", rowsAffected)
	return nil
}

// DeleteTx удаляет запись в транзакции
func (r *Repository) DeleteTx(ctx context.Context, tx persistence.Transaction, id int64) error {
	query := `DELETE FROM test WHERE id = $1`
	rowsAffected, err := tx.ExecWithResult(ctx, query, id)
	if err != nil {
		r.Log.Error("failed to delete test in transaction", "error", err, "id", id)
		return fmt.Errorf("failed to delete test in transaction: %w", err)
	}

	if rowsAffected == 0 {
		r.Log.Warn("test not found for delete in transaction", "id", id)
		return fmt.Errorf("test not found")
	}

	r.Log.Debug("test deleted in transaction", "id", id, "rowsAffected", rowsAffected)
	return nil
}

// GetByIDTx получает запись по ID в транзакции
func (r *Repository) GetByIDTx(ctx context.Context, tx persistence.Transaction, id int64) (*domain.Test, error) {
	var test domain.Test
	query := `SELECT id, filed1, filed2 FROM test WHERE id = $1`
	err := tx.Get(ctx, &test, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.Log.Warn("test not found in transaction", "id", id)
			return nil, fmt.Errorf("test not found: %w", err)
		}
		r.Log.Error("failed to get test in transaction", "error", err, "id", id)
		return nil, fmt.Errorf("failed to get test in transaction: %w", err)
	}
	r.Log.Debug("test retrieved in transaction", "id", id)
	return &test, nil
}
