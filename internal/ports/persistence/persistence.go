package persistence

import "github.com/jmoiron/sqlx"

type Persistence interface {
	Get(dest interface{}, query string, args ...interface{}) error
	Select(dest interface{}, query string, args ...interface{}) error
	Exec(query string, args ...interface{}) error
	ExecWithResult(query string, args ...interface{}) (int64, error)
	NamedExec(query string, arg interface{}) error
	NamedExecWithResult(query string, arg interface{}) (int64, error)
	QueryRow(query string, args ...interface{}) *sqlx.Row
	NamedQuery(query string, arg interface{}) (*sqlx.Rows, error)
}
