package persistence

import (
	"context"

	"github.com/jmoiron/sqlx"
)

// Persistence интерфейс для работы с базой данных (no tx)
type Persistence interface {
	Get(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	Select(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	Exec(ctx context.Context, query string, args ...interface{}) error
	ExecWithResult(ctx context.Context, query string, args ...interface{}) (int64, error)
	NamedExec(ctx context.Context, query string, arg interface{}) error
	NamedExecWithResult(ctx context.Context, query string, arg interface{}) (int64, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) *sqlx.Row
	NamedQuery(ctx context.Context, query string, arg interface{}) (*sqlx.Rows, error)

	// Управление транзакциями
	BeginTx(ctx context.Context) (Transaction, error)
	WithTransaction(ctx context.Context, fn func(context.Context, Transaction) error) error
}

// Transaction интерфейс для работы с транзакциями
type Transaction interface {
	Get(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	Select(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	Exec(ctx context.Context, query string, args ...interface{}) error
	ExecWithResult(ctx context.Context, query string, args ...interface{}) (int64, error)
	NamedExec(ctx context.Context, query string, arg interface{}) error
	NamedExecWithResult(ctx context.Context, query string, arg interface{}) (int64, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) *sqlx.Row
	NamedQuery(ctx context.Context, query string, arg interface{}) (*sqlx.Rows, error)

	// Управление транзакцией
	Commit() error
	Rollback() error
}
