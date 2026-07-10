package services

import (
	"context"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
)

type OperatorService struct {
	repo interfaces.OperatorRepository
}

func NewOperatorService(repo interfaces.OperatorRepository) *OperatorService {
	return &OperatorService{repo: repo}
}

func (s *OperatorService) List(ctx context.Context) ([]domain.Operator, error) {
	return s.repo.List(ctx)
}

func (s *OperatorService) Save(ctx context.Context, operator *domain.Operator) error {
	return s.repo.Create(ctx, operator)
}