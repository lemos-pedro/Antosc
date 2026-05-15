package services

import (
	"context"
	"errors"
	"strings"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
)

type AuditService struct {
	repo interfaces.AuditRepository
}

func NewAuditService(repo interfaces.AuditRepository) *AuditService {
	return &AuditService{repo: repo}
}

func (s *AuditService) List(ctx context.Context, filter interfaces.AuditFilter) ([]domain.AuditLog, int, error) {
	if filter.Limit <= 0 || filter.Limit > 200 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	return s.repo.List(ctx, filter)
}

func (s *AuditService) GetByID(ctx context.Context, id string) (*domain.AuditLog, error) {
	if strings.TrimSpace(id) == "" {
		return nil, errors.New("audit_id is required")
	}
	return s.repo.GetByID(ctx, id)
}
