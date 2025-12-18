package pg

import (
	"context"

	"github.com/admin/tg-bots/astro-bot/internal/ports/persistence"
	"github.com/jmoiron/sqlx"
)

// Tx обёртка над sqlx.Tx для работы с транзакциями
// Реализует только Transaction, НЕ Persistence
// ctx сохраняется при создании транзакции для логирования/метрик,
// но в методах используется переданный контекст для гибкости
type Tx struct {
	tx  *sqlx.Tx
	ctx context.Context // сохранённый контекст создания транзакции (для логирования/метрик)
}

// Проверка, что Tx реализует persistence.Transaction
var _ persistence.Transaction = (*Tx)(nil)

// Get выполняет запрос в транзакции и сканирует результат в структуру
// Использует переданный ctx для гибкости (timeout, cancellation)
func (t *Tx) Get(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return t.tx.GetContext(ctx, dest, query, args...)
}

// Select выполняет запрос в транзакции и сканирует результаты в слайс
// Использует переданный ctx для гибкости (timeout, cancellation)
func (t *Tx) Select(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return t.tx.SelectContext(ctx, dest, query, args...)
}

// Exec выполняет запрос в транзакции без возврата данных
// Использует переданный ctx для гибкости (timeout, cancellation)
func (t *Tx) Exec(ctx context.Context, query string, args ...interface{}) error {
	_, err := t.tx.ExecContext(ctx, query, args...)
	return err
}

// ExecWithResult выполняет запрос в транзакции и возвращает количество затронутых строк
// Использует переданный ctx для гибкости (timeout, cancellation)
func (t *Tx) ExecWithResult(ctx context.Context, query string, args ...interface{}) (int64, error) {
	result, err := t.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// NamedExec выполняет именованный запрос в транзакции
// Использует переданный ctx для гибкости (timeout, cancellation)
func (t *Tx) NamedExec(ctx context.Context, query string, arg interface{}) error {
	_, err := t.tx.NamedExecContext(ctx, query, arg)
	return err
}

// NamedExecWithResult выполняет именованный запрос в транзакции и возвращает количество затронутых строк
// Использует переданный ctx для гибкости (timeout, cancellation)
func (t *Tx) NamedExecWithResult(ctx context.Context, query string, arg interface{}) (int64, error) {
	result, err := t.tx.NamedExecContext(ctx, query, arg)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// QueryRow выполняет запрос в транзакции и возвращает строку для сканирования
// Использует переданный ctx для гибкости (timeout, cancellation)
func (t *Tx) QueryRow(ctx context.Context, query string, args ...interface{}) *sqlx.Row {
	return t.tx.QueryRowxContext(ctx, query, args...)
}

// NamedQuery выполняет именованный запрос в транзакции и возвращает строки
// Примечание: sqlx.Tx не имеет NamedQueryContext, поэтому контекст не используется
// Для контроля timeout используйте context.WithTimeout перед вызовом BeginTx
func (t *Tx) NamedQuery(ctx context.Context, query string, arg interface{}) (*sqlx.Rows, error) {
	return t.tx.NamedQuery(query, arg)
}

// Commit фиксирует транзакцию
func (t *Tx) Commit() error {
	return t.tx.Commit()
}

// Rollback откатывает транзакцию
func (t *Tx) Rollback() error {
	return t.tx.Rollback()
}
