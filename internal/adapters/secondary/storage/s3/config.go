package s3

import (
	"context"
	"fmt"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Config struct {
	Host      string `envconfig:"HOST" required:"true"`        // localhost:9000
	AccessKey string `envconfig:"ACCESS_KEY" required:"true"`  // minioadmin
	SecretKey string `envconfig:"SECRET_KEY" required:"true"` // minioadmin
	Bucket    string `envconfig:"BUCKET" default:"images"`     // images
	UseSSL    bool   `envconfig:"USE_SSL" default:"false"`     // false для локальной разработки
}

// NewClient создаёт новый MinIO клиент
func (c *Config) NewClient() (*minio.Client, error) {
	client, err := minio.New(c.Host, &minio.Options{
		Creds:  credentials.NewStaticV4(c.AccessKey, c.SecretKey, ""),
		Secure: c.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	// Проверяем подключение
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Проверяем существование bucket
	exists, err := client.BucketExists(ctx, c.Bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		return nil, fmt.Errorf("bucket %s does not exist", c.Bucket)
	}

	return client, nil
}
