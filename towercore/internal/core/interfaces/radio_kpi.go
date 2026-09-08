// Package interfaces contém contratos (interfaces) que definem o comportamento
// esperado de varios componentes do sistema.
package interfaces

import (
	"context"
	"time"

	"github.com/google/uuid"
	"towercore/internal/core/domain"
)

// RadioKPIGenericRepository define a interface genérica para repositórios de KPIs de rádio.
// Esta abstração permite trocar facilmente a implementação (PostgreSQL, in-memory para teste, etc.).
type RadioKPIGenericRepository interface {
	// CreateMany insere múltiplos KPIs de rádio de uma vez (ótimo para batch de ETLs)
	CreateMany(ctx context.Context, kpis []*domain.RadioKPI) error
	// List retorna KPIs de rádio baseado em filtros
	List(ctx context.Context, filter *domain.RadioKPIFilter) ([]*domain.RadioKPI, int, error)
	// DeleteOlderThan remove KPIs mais antigos que um determinado tempo (para retenção)
	DeleteOlderThan(ctx context.Context, olderThan time.Time) error
	// GetByTowerAndSector retorna o KPI mais recente para uma torre/setor/técnica específica
	GetByTowerAndSector(ctx context.Context, towerID uuid.UUID, sectorID string, technique string) (*domain.RadioKPI, error)
}
