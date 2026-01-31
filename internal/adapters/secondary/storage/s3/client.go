package s3

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/admin/tg-bots/astro-bot/internal/ports/storage"
	"github.com/minio/minio-go/v7"
	"log/slog"
)

// Client обёртка над minio.Client для работы с S3
type Client struct {
	client *minio.Client
	bucket string
	log    *slog.Logger
}

// NewClient создаёт новый S3 клиент
func NewClient(client *minio.Client, bucket string, log *slog.Logger) storage.IS3Client {
	return &Client{
		client: client,
		bucket: bucket,
		log:    log,
	}
}

// GetFile получает файл по пути
func (c *Client) GetFile(ctx context.Context, path string) ([]byte, error) {
	object, err := c.client.GetObject(ctx, c.bucket, path, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object %s: %w", path, err)
	}
	defer object.Close()

	data, err := io.ReadAll(object)
	if err != nil {
		return nil, fmt.Errorf("failed to read object %s: %w", path, err)
	}

	return data, nil
}

// ListFiles получает список файлов по префиксу
func (c *Client) ListFiles(ctx context.Context, prefix string) ([]string, error) {
	var files []string

	objectCh := c.client.ListObjects(ctx, c.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("failed to list objects with prefix %s: %w", prefix, object.Err)
		}

		// Пропускаем директории (объекты, заканчивающиеся на /)
		if object.Key[len(object.Key)-1] != '/' {
			files = append(files, object.Key)
		}
	}

	return files, nil
}

// GetPresignedURL генерирует presigned URL для файла
func (c *Client) GetPresignedURL(ctx context.Context, path string, expires time.Duration) (string, error) {
	if expires <= 0 {
		expires = 5 * time.Minute // дефолтный TTL
	}

	url, err := c.client.PresignedGetObject(ctx, c.bucket, path, expires, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL for %s: %w", path, err)
	}

	return url.String(), nil
}
