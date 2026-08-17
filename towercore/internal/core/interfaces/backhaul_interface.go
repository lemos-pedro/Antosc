// Package interfaces contém contratos (interfaces) que definem o comportamento
// esperado de varios componentes do sistema.
package interfaces

import (
	"context"
	"time"

	"towercore/internal/core/domain"
)

// BackhaulInterfaceRepository define a interface para repositórios de métricas de backhaul.
type BackhaulInterfaceRepository interface {
	// Create insere uma nova medição de interface de backhaul
	Create(ctx context.Context, iface *domain.BackhaulInterface) error
	// CreateMany insere múltiplas medições de uma vez (ótimo para batch)
	CreateMany(ctx context.Context, ifaces []*domain.BackhaulInterface) error
	// List retorna medições baseado em filtros
	List(ctx context.Context, filter *domain.BackhaulInterfaceFilter) ([]*domain.BackhaulInterface, int, error)
	// GetLatest retorna a medição mais recente para uma torre/interface
	GetLatest(ctx context.Context, towerID uuid.UUID, interfaceName string) (*domain.BackhaulInterface, error)
	// DeleteOlderThan remove medições mais antigas que um determinado tempo (para retenção)
	DeleteOlderThan(ctx context.Context, olderThan time.Time) error
}