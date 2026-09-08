package interfaces

import (
	"context"

	"towercore/internal/core/domain"
)

type OperatorRepository interface {
	List(ctx context.Context) ([]domain.Operator, error)
	Create(ctx context.Context, operator *domain.Operator) error
	GetByID(ctx context.Context, operatorID string) (*domain.Operator, error)
	Update(ctx context.Context, operator *domain.Operator) error
	Delete(ctx context.Context, operatorID string) error
}
