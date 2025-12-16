package repository

import "github.com/admin/tg-bots/astro-bot/internal/domain"

type ITestRepo interface {
	Create(test *domain.Test) error
	GetByID(id int64) (*domain.Test, error)
	GetAll() ([]*domain.Test, error)
	Update(test *domain.Test) error
	Delete(id int64) error
}
