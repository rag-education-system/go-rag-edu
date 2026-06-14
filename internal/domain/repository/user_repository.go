package repository

import (
	"context"
	"rag-api/internal/domain/entity"
)

type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	FindById(ctx context.Context, id string) (*entity.User, error)
	ListAll(ctx context.Context) ([]entity.User, error)
}
