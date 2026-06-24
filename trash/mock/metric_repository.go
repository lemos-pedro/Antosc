package mock

import (
	"context"
	"slices"
	"sync"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
)

type MetricRepository struct {
	mu      sync.Mutex
	metrics []domain.Metric
}

func NewMetricRepository() *MetricRepository {
	return &MetricRepository{}
}

func (r *MetricRepository) Create(_ context.Context, metric *domain.Metric) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.metrics = append(r.metrics, *metric)
	return nil
}

func (r *MetricRepository) List(_ context.Context, filter interfaces.MetricFilter) ([]domain.Metric, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	filtered := make([]domain.Metric, 0, len(r.metrics))
	for _, m := range r.metrics {
		if filter.TowerID != "" && m.TowerID != filter.TowerID {
			continue
		}
		if filter.From != nil && m.CollectedAt.Before(*filter.From) {
			continue
		}
		if filter.To != nil && m.CollectedAt.After(*filter.To) {
			continue
		}
		filtered = append(filtered, m)
	}

	total := len(filtered)
	start := min(filter.Offset, total)
	end := min(start+filter.Limit, total)
	return slices.Clone(filtered[start:end]), total, nil
}
