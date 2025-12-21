package pg

import (
	"context"

	"github.com/admin/tg-bots/astro-bot/internal/ports/persistence"
	"github.com/jmoiron/sqlx"
)

// DB обёртка над sqlx.DB для работы с базой данных
// Реализует только Persistence, НЕ Transaction
type DB struct {
	Db *sqlx.DB
}

func NewDB(db *sqlx.DB) *DB {
	return &DB{Db: db}
}

// Get выполняет запрос и сканирует результат в структуру (одна запись)
func (d *DB) Get(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return d.Db.GetContext(ctx, dest, query, args...)
}

// Select выполняет запрос и сканирует результаты в слайс структур
func (d *DB) Select(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return d.Db.SelectContext(ctx, dest, query, args...)
}

// Exec выполняет запрос без возврата данных (INSERT, UPDATE, DELETE)
func (d *DB) Exec(ctx context.Context, query string, args ...interface{}) error {
	_, err := d.Db.ExecContext(ctx, query, args...)
	return err
}

// ExecWithResult выполняет запрос и возвращает количество затронутых строк
func (d *DB) ExecWithResult(ctx context.Context, query string, args ...interface{}) (int64, error) {
	result, err := d.Db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// NamedExec выполняет именованный запрос (использует struct tags)
func (d *DB) NamedExec(ctx context.Context, query string, arg interface{}) error {
	_, err := d.Db.NamedExecContext(ctx, query, arg)
	return err
}

// NamedExecWithResult выполняет именованный запрос и возвращает количество затронутых строк
func (d *DB) NamedExecWithResult(ctx context.Context, query string, arg interface{}) (int64, error) {
	result, err := d.Db.NamedExecContext(ctx, query, arg)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// QueryRow выполняет запрос и возвращает строку для сканирования
func (d *DB) QueryRow(ctx context.Context, query string, args ...interface{}) *sqlx.Row {
	return d.Db.QueryRowxContext(ctx, query, args...)
}

// NamedQuery выполняет именованный запрос и возвращает строки
func (d *DB) NamedQuery(ctx context.Context, query string, arg interface{}) (*sqlx.Rows, error) {
	return d.Db.NamedQuery(query, arg)
}

// BeginTx начинает новую транзакцию
func (d *DB) BeginTx(ctx context.Context) (persistence.Transaction, error) {
	tx, err := d.Db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &Tx{tx: tx, ctx: ctx}, nil
}

// WithTransaction выполняет функцию в транзакции с автоматическим commit/rollback
func (d *DB) WithTransaction(ctx context.Context, fn func(context.Context, persistence.Transaction) error) error {
	tx, err := d.BeginTx(ctx)
	if err != nil {
		return err
	}

	if err := fn(ctx, tx); err != nil {
		// Если ошибка - откатываем
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return rollbackErr
		}
		return err
	}

	return tx.Commit()
}

// Close закрывает подключение к базе данных
func (d *DB) Close() error {
	return d.Db.Close()
}
