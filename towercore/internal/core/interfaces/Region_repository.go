package interfaces

import (
	"context"

	"towercore/internal/core/domain"
)

type RegionRepository interface {
	List(ctx context.Context) ([]domain.Region, error)
	Create(ctx context.Context, region *domain.Region) error
	GetByID(ctx context.Context, regionID string) (*domain.Region, error)
}