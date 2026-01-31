package storage

import (
	"context"
	"time"
)

// IS3Client интерфейс для работы с S3-совместимым хранилищем (MinIO)
type IS3Client interface {
	GetFile(ctx context.Context, path string) ([]byte, error)
	ListFiles(ctx context.Context, prefix string) ([]string, error)
	GetPresignedURL(ctx context.Context, path string, expires time.Duration) (string, error)
}
