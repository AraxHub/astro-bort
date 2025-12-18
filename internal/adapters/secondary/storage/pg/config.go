package pg

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

const (
	maxOpenConnections            = 25
	maxIdleConnections            = 5
	connMaxLifetime               = 5 * time.Minute
	connMaxIdleTime               = 1 * time.Minute
	defaultStatementTimeoutMillis = 60000
)

type Config struct {
	Host                   string `envconfig:"HOST"`
	Port                   string `envconfig:"PORT"`
	Username               string `envconfig:"USERNAME"`
	Password               string `envconfig:"PASSWORD"`
	Database               string `envconfig:"DATABASE"`
	SSLMode                string `envconfig:"SSL_MODE"`
	StatementTimeoutMillis int    `envconfig:"STATEMENT_TIMEOUT" default:"60000"`
}

func (c *Config) toPgConnection() string {
	return fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s",
		c.Host,
		c.Port,
		c.Username,
		c.Database,
		c.Password,
		c.SSLMode,
	)
}

// NewConnection создает новое подключение к базе данных с настройками пула и statement_timeout
func (c *Config) NewConnection() (*sqlx.DB, error) {
	connectionConfig, err := pgx.ParseConfig(c.toPgConnection())
	if err != nil {
		return nil, fmt.Errorf("parse db config: %w", err)
	}

	connectionString := stdlib.RegisterConnConfig(connectionConfig)
	db, err := sqlx.Connect("pgx", connectionString)
	if err != nil {
		return nil, fmt.Errorf("connect db error: %w", err)
	}

	db.SetMaxOpenConns(maxOpenConnections)
	db.SetConnMaxLifetime(connMaxLifetime)
	db.SetMaxIdleConns(maxIdleConnections)
	db.SetConnMaxIdleTime(connMaxIdleTime)

	if err = db.Ping(); err != nil {
		return nil, err
	}

	timeout := c.StatementTimeoutMillis
	if timeout <= 0 {
		timeout = defaultStatementTimeoutMillis
	}
	_, err = db.Exec(fmt.Sprintf("SET statement_timeout = %d", timeout))
	if err != nil {
		return nil, fmt.Errorf("set statement_timeout failed: %w", err)
	}

	return db, err
}
