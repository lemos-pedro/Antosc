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

	// LastCollectedAt devolve o timestamp da métrica mais recente
	// recebida para a torre, ou nil se nunca houve nenhuma. Usado por
	// AvailabilityService para detetar downtime por ausência de sinal
	// (torre sem heartbeat/pipeline de coleta interrompido), não só por
	// eventos type=failure explícitos que dependem do ingest service
	// conseguir detetar e registar a falha.
	LastCollectedAt(ctx context.Context, towerID string) (*time.Time, error)
}
