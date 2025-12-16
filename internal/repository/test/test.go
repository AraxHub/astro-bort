package testRepo

import (
	"database/sql"
	"fmt"

	"github.com/admin/tg-bots/astro-bot/internal/adapters/secondary/storage/pg"
	ports "github.com/admin/tg-bots/astro-bot/internal/ports/repository"
	"github.com/jmoiron/sqlx"

	"log/slog"

	"github.com/admin/tg-bots/astro-bot/internal/domain"
	"github.com/admin/tg-bots/astro-bot/internal/ports/persistence"
)

type Repository struct {
	db     persistence.Persistence
	logger *slog.Logger
}

func New(db *sqlx.DB, logger *slog.Logger) ports.ITestRepo {
	return &Repository{
		db:     pg.NewPgPersistence(db),
		logger: logger,
	}
}

// Create создает новую запись в таблице test
func (r *Repository) Create(test *domain.Test) error {
	query := `INSERT INTO test (filed1, filed2) VALUES ($1, $2) RETURNING id`
	err := r.db.QueryRow(query, test.Filed1, test.Filed2).Scan(&test.ID)
	if err != nil {
		r.logger.Error("failed to create test", "error", err, "filed1", test.Filed1, "filed2", test.Filed2)
		return fmt.Errorf("failed to create test: %w", err)
	}
	r.logger.Debug("test created successfully", "id", test.ID)
	return nil
}

// GetByID получает запись по ID
func (r *Repository) GetByID(id int64) (*domain.Test, error) {
	var test domain.Test
	query := `SELECT id, filed1, filed2 FROM test WHERE id = $1`
	err := r.db.Get(&test, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			r.logger.Warn("test not found", "id", id)
			return nil, fmt.Errorf("test not found: %w", err)
		}
		r.logger.Error("failed to get test", "error", err, "id", id)
		return nil, fmt.Errorf("failed to get test: %w", err)
	}
	r.logger.Debug("test retrieved successfully", "id", id)
	return &test, nil
}

// GetAll получает все записи из таблицы test
func (r *Repository) GetAll() ([]*domain.Test, error) {
	var tests []*domain.Test
	query := `SELECT id, filed1, filed2 FROM test ORDER BY id`
	err := r.db.Select(&tests, query)
	if err != nil {
		r.logger.Error("failed to get all tests", "error", err)
		return nil, fmt.Errorf("failed to get all tests: %w", err)
	}
	r.logger.Debug("tests retrieved successfully", "count", len(tests))
	return tests, nil
}

// Update обновляет запись в таблице test
func (r *Repository) Update(test *domain.Test) error {
	query := `UPDATE test SET filed1 = :filed1, filed2 = :filed2 WHERE id = :id`
	rowsAffected, err := r.db.NamedExecWithResult(query, test)
	if err != nil {
		r.logger.Error("failed to update test", "error", err, "id", test.ID)
		return fmt.Errorf("failed to update test: %w", err)
	}

	if rowsAffected == 0 {
		r.logger.Warn("test not found for update", "id", test.ID)
		return fmt.Errorf("test not found")
	}

	r.logger.Debug("test updated successfully", "id", test.ID, "rowsAffected", rowsAffected)
	return nil
}

// Delete удаляет запись по ID
func (r *Repository) Delete(id int64) error {
	query := `DELETE FROM test WHERE id = $1`
	rowsAffected, err := r.db.ExecWithResult(query, id)
	if err != nil {
		r.logger.Error("failed to delete test", "error", err, "id", id)
		return fmt.Errorf("failed to delete test: %w", err)
	}

	if rowsAffected == 0 {
		r.logger.Warn("test not found for delete", "id", id)
		return fmt.Errorf("test not found")
	}

	r.logger.Debug("test deleted successfully", "id", id, "rowsAffected", rowsAffected)
	return nil
}
