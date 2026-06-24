package interfaces

import (
	"context"

	"towercore/internal/core/domain"
)

type UserRepository interface {
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	Upsert(ctx context.Context, user *domain.User) error
	Create(ctx context.Context, user *domain.User) error
}