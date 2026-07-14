package towercore

import (
	"context"
	"time"
)

// Client é a porta de saída do AIP para o towercore. Só leitura — o AIP
// nunca escreve no towercore (sem controlo autónomo de equipamento).
type Client interface {
	GetTowers(ctx context.Context) ([]TowerDTO, error)
	GetMetrics(ctx context.Context, since time.Time) ([]MetricDTO, error)
	GetEvents(ctx context.Context) ([]EventDTO, error)
}
