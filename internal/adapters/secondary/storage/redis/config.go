package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	defaultMaxRetries      = 3
	defaultDialTimeout     = 5 * time.Second
	defaultReadTimeout     = 3 * time.Second
	defaultWriteTimeout    = 3 * time.Second
	defaultPoolSize        = 10
	defaultMinIdleConns    = 5
	defaultConnMaxLifetime = 30 * time.Minute
	defaultConnMaxIdleTime = 5 * time.Minute
)

type Config struct {
	Host            string `envconfig:"HOST" default:"localhost"`
	Port            string `envconfig:"PORT" default:"6379"`
	Username        string `envconfig:"USERNAME" required:"true"`
	Password        string `envconfig:"PASSWORD"`
	Database        int    `envconfig:"DATABASE" default:"0"`
	MaxRetries      int    `envconfig:"MAX_RETRIES" default:"3"`
	DialTimeout     int    `envconfig:"DIAL_TIMEOUT" default:"5"`  // в секундах
	ReadTimeout     int    `envconfig:"READ_TIMEOUT" default:"3"`  // в секундах
	WriteTimeout    int    `envconfig:"WRITE_TIMEOUT" default:"3"` // в секундах
	PoolSize        int    `envconfig:"POOL_SIZE" default:"10"`
	MinIdleConns    int    `envconfig:"MIN_IDLE_CONNS" default:"5"`
	ConnMaxLifetime int    `envconfig:"CONN_MAX_LIFETIME" default:"30"` // в минутах
	ConnMaxIdleTime int    `envconfig:"CONN_MAX_IDLE_TIME" default:"5"` // в минутах
}

// NewConnection создаёт новое подключение к Redis
func (c *Config) NewConnection() (*redis.Client, error) {
	dialTimeout := time.Duration(c.DialTimeout) * time.Second
	if dialTimeout <= 0 {
		dialTimeout = defaultDialTimeout
	}

	readTimeout := time.Duration(c.ReadTimeout) * time.Second
	if readTimeout <= 0 {
		readTimeout = defaultReadTimeout
	}

	writeTimeout := time.Duration(c.WriteTimeout) * time.Second
	if writeTimeout <= 0 {
		writeTimeout = defaultWriteTimeout
	}

	poolSize := c.PoolSize
	if poolSize <= 0 {
		poolSize = defaultPoolSize
	}

	minIdleConns := c.MinIdleConns
	if minIdleConns <= 0 {
		minIdleConns = defaultMinIdleConns
	}

	connMaxLifetime := time.Duration(c.ConnMaxLifetime) * time.Minute
	if connMaxLifetime <= 0 {
		connMaxLifetime = defaultConnMaxLifetime
	}

	connMaxIdleTime := time.Duration(c.ConnMaxIdleTime) * time.Minute
	if connMaxIdleTime <= 0 {
		connMaxIdleTime = defaultConnMaxIdleTime
	}

	maxRetries := c.MaxRetries
	if maxRetries < 0 {
		maxRetries = defaultMaxRetries
	}

	addr := fmt.Sprintf("%s:%s", c.Host, c.Port)

	rdb := redis.NewClient(&redis.Options{
		Addr:            addr,
		Username:        c.Username,
		Password:        c.Password,
		DB:              c.Database,
		MaxRetries:      maxRetries,
		DialTimeout:     dialTimeout,
		ReadTimeout:     readTimeout,
		WriteTimeout:    writeTimeout,
		PoolSize:        poolSize,
		MinIdleConns:    minIdleConns,
		ConnMaxLifetime: connMaxLifetime,
		ConnMaxIdleTime: connMaxIdleTime,
	})

	// Проверяем подключение
	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return rdb, nil
}
