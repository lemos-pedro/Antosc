package services

import (
	"context"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
)

type RegionService struct {
	repo interfaces.RegionRepository
}

func NewRegionService(repo interfaces.RegionRepository) *RegionService {
	return &RegionService{repo: repo}
}

func (s *RegionService) List(ctx context.Context) ([]domain.Region, error) {
	return s.repo.List(ctx)
}

func (s *RegionService) Save(ctx context.Context, region *domain.Region) error {
	return s.repo.Create(ctx, region)
}