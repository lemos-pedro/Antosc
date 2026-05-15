package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
)

type MetricService struct {
	repo interfaces.MetricRepository
}

func NewMetricService(repo interfaces.MetricRepository) *MetricService {
	return &MetricService{repo: repo}
}

func (s *MetricService) Create(ctx context.Context, metric *domain.Metric) error {
	if strings.TrimSpace(metric.TowerID) == "" {
		return errors.New("tower_id is required")
	}
	if len(metric.Values) == 0 {
		return errors.New("metrics is required")
	}

	if metric.ID == "" {
		id, err := newUUIDv4()
		if err != nil {
			return err
		}
		metric.ID = id
	}

	now := time.Now().UTC()
	if metric.CollectedAt.IsZero() {
		metric.CollectedAt = now
	}
	metric.CreatedAt = now

	return s.repo.Create(ctx, metric)
}

func (s *MetricService) List(ctx context.Context, filter interfaces.MetricFilter) ([]domain.Metric, int, error) {
	if filter.Limit <= 0 || filter.Limit > 200 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	return s.repo.List(ctx, filter)
}
