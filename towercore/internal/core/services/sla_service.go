package services

import (
	"context"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
)

type SLAService struct {
	repo interfaces.SLARepository
}

func NewSLAService(repo interfaces.SLARepository) *SLAService {
	return &SLAService{repo: repo}
}

func (s *SLAService) GetGlobal(ctx context.Context) (*domain.SLA, error) {
	return s.repo.GetGlobal(ctx)
}
func (s *SLAService) GetByRegion(ctx context.Context, regionID string) (*domain.SLA, error) {
	return s.repo.GetByRegion(ctx, regionID)
}