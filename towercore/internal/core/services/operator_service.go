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

func (s *OperatorService) GetByID(ctx context.Context, id string) (*domain.Operator, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *OperatorService) Update(ctx context.Context, operator *domain.Operator) error {
	return s.repo.Update(ctx, operator)
}

func (s *OperatorService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}