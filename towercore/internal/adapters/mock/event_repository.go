package mock

import (
	"context"
	"slices"
	"sync"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
)

type EventRepository struct {
	mu     sync.Mutex
	events []domain.Event
}

func NewEventRepository() *EventRepository {
	return &EventRepository{}
}

func (r *EventRepository) Create(_ context.Context, event *domain.Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, *event)
	return nil
}

func (r *EventRepository) List(_ context.Context, filter interfaces.EventFilter) ([]domain.Event, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	filtered := make([]domain.Event, 0, len(r.events))
	for _, ev := range r.events {
		if filter.TowerID != "" && ev.TowerID != filter.TowerID {
			continue
		}
		if filter.Type != "" && string(ev.Type) != filter.Type {
			continue
		}
		if filter.Severity != "" && string(ev.Severity) != filter.Severity {
			continue
		}
		filtered = append(filtered, ev)
	}

	total := len(filtered)
	start := min(filter.Offset, total)
	end := min(start+filter.Limit, total)
	return slices.Clone(filtered[start:end]), total, nil
}
