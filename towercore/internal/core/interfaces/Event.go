package interfaces

import (
	"context"

	"towercore/internal/core/domain"
)

type EventFilter struct {
	TowerID  string
	Type     string
	Severity string
	Limit    int
	Offset   int
}

type EventRepository interface {
	Create(ctx context.Context, event *domain.Event) error
	List(ctx context.Context, filter EventFilter) ([]domain.Event, int, error)
}
