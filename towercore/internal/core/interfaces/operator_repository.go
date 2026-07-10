package interfaces

import (
	"context"

	"towercore/internal/core/domain"
)

type OperatorRepository interface {
	List(ctx context.Context) ([]domain.Operator, error)
	Create(ctx context.Context, operator *domain.Operator) error
}