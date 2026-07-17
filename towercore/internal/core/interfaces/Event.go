package interfaces

import (
	"context"
	"errors"
	"time"

	"towercore/internal/core/domain"
)

var ErrEventNotFound = errors.New("event not found")

type EventFilter struct {
	TowerID  string
	Type     string
	Severity string
	Status   string // "open" | "resolved" | "" (todos)
	Limit    int
	Offset   int
}

type EventRepository interface {
	Create(ctx context.Context, event *domain.Event) error
	List(ctx context.Context, filter EventFilter) ([]domain.Event, int, error)

	// Deduplicação de alarmes: evita criar um evento novo a cada ciclo de
	// poll enquanto a mesma condição de alarme continuar ativa.
	FindOpenByTowerAndAlarmKey(ctx context.Context, towerID, alarmKey string) (*domain.Event, error)
	TouchLastSeen(ctx context.Context, eventID string, ts time.Time) error
	ResolveOpenByTowerAndAlarmKey(ctx context.Context, towerID, alarmKey string, resolvedAt time.Time) error
	SumFailureDowntime(ctx context.Context, towerID string, windowStart, windowEnd time.Time) (float64, error)
}