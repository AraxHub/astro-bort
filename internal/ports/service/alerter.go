package service

import (
	"context"
)

// IAlerterService интерфейс для отправки алертов
type IAlerterService interface {
	SendAlert(ctx context.Context, message string) error
}
