package pg

import (
	"context"

	"github.com/jmoiron/sqlx"
)

// Tx обёртка над sqlx.Tx для работы с транзакциями
// ctx сохраняется при создании транзакции для логирования/метрик, в методах используется переданный контекст
type Tx struct {
	tx  *sqlx.Tx
	ctx context.Context // сохранённый контекст создания транзакции (для логирования/метрик)
}

// Get запрос в транзакции и сканирует результат в структуру
func (t *Tx) Get(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return t.tx.GetContext(ctx, dest, query, args...)
}

// Select запрос в транзакции и сканирует результаты в слайс
func (t *Tx) Select(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return t.tx.SelectContext(ctx, dest, query, args...)
}

// Exec запрос в транзакции без возврата данных
func (t *Tx) Exec(ctx context.Context, query string, args ...interface{}) error {
	_, err := t.tx.ExecContext(ctx, query, args...)
	return err
}

// ExecWithResult запрос в транзакции и возвращает количество затронутых строк
func (t *Tx) ExecWithResult(ctx context.Context, query string, args ...interface{}) (int64, error) {
	result, err := t.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// NamedExec именованный запрос в транзакции
func (t *Tx) NamedExec(ctx context.Context, query string, arg interface{}) error {
	_, err := t.tx.NamedExecContext(ctx, query, arg)
	return err
}

// NamedExecWithResult именованный запрос в транзакции и возвращает количество затронутых строк
func (t *Tx) NamedExecWithResult(ctx context.Context, query string, arg interface{}) (int64, error) {
	result, err := t.tx.NamedExecContext(ctx, query, arg)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// QueryRow запрос в транзакции и возвращает строку для сканирования
func (t *Tx) QueryRow(ctx context.Context, query string, args ...interface{}) *sqlx.Row {
	return t.tx.QueryRowxContext(ctx, query, args...)
}

// NamedQuery выполняет именованный запрос в транзакции и возвращает строки
// Примечание: sqlx.Tx не имеет NamedQueryContext, поэтому контекст не используется
func (t *Tx) NamedQuery(ctx context.Context, query string, arg interface{}) (*sqlx.Rows, error) {
	return t.tx.NamedQuery(query, arg)
}

func (t *Tx) Commit() error {
	return t.tx.Commit()
}

func (t *Tx) Rollback() error {
	return t.tx.Rollback()
}
