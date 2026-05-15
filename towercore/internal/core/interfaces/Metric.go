package interfaces

import (
	"context"
	"time"

	"towercore/internal/core/domain"
)

type MetricFilter struct {
	TowerID string
	From    *time.Time
	To      *time.Time
	Limit   int
	Offset  int
}

type MetricRepository interface {
	Create(ctx context.Context, metric *domain.Metric) error
	List(ctx context.Context, filter MetricFilter) ([]domain.Metric, int, error)
}
