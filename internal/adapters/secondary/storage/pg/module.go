package pg

import (
	"github.com/jmoiron/sqlx"
)

// Get выполняет запрос и сканирует результат в структуру (одна запись)
// Автоматически использует struct tags для маппинга
// Пример: db.Get(&user, "SELECT * FROM users WHERE id = $1", userID)
func (d *DB) Get(dest interface{}, query string, args ...interface{}) error {
	return d.Db.Get(dest, query, args...)
}

// Select выполняет запрос и сканирует результаты в слайс структур
// Автоматически использует struct tags для маппинга
// Пример: db.Select(&users, "SELECT * FROM users WHERE age > $1", 18)
func (d *DB) Select(dest interface{}, query string, args ...interface{}) error {
	return d.Db.Select(dest, query, args...)
}

// Exec выполняет запрос без возврата данных (INSERT, UPDATE, DELETE)
// Пример: db.Exec("DELETE FROM users WHERE id = $1", userID)
func (d *DB) Exec(query string, args ...interface{}) error {
	_, err := d.Db.Exec(query, args...)
	return err
}

// ExecWithResult выполняет запрос и возвращает количество затронутых строк
// Пример: rows, _ := db.ExecWithResult("UPDATE users SET name = $1 WHERE id = $2", "John", userID)
func (d *DB) ExecWithResult(query string, args ...interface{}) (int64, error) {
	result, err := d.Db.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// NamedExec выполняет именованный запрос (использует struct tags)
// Пример: db.NamedExec("INSERT INTO users (name, email) VALUES (:name, :email)", user)
func (d *DB) NamedExec(query string, arg interface{}) error {
	_, err := d.Db.NamedExec(query, arg)
	return err
}

// NamedExecWithResult выполняет именованный запрос и возвращает количество затронутых строк
// Пример: rows, _ := db.NamedExecWithResult("UPDATE users SET name = :name WHERE id = :id", user)
func (d *DB) NamedExecWithResult(query string, arg interface{}) (int64, error) {
	result, err := d.Db.NamedExec(query, arg)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// QueryRow выполняет запрос и возвращает строку для сканирования
// Используется для запросов с RETURNING
// Пример: db.QueryRow("INSERT INTO users (name) VALUES ($1) RETURNING id", "John").Scan(&id)
func (d *DB) QueryRow(query string, args ...interface{}) *sqlx.Row {
	return d.Db.QueryRowx(query, args...)
}

// NamedQuery выполняет именованный запрос и возвращает строки
// Пример: rows, _ := db.NamedQuery("SELECT * FROM users WHERE name = :name", map[string]interface{}{"name": "John"})
func (d *DB) NamedQuery(query string, arg interface{}) (*sqlx.Rows, error) {
	return d.Db.NamedQuery(query, arg)
}
