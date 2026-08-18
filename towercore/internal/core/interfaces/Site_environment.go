// Package interfaces contém contratos (interfaces) que definem o comportamento
// esperado de varios componentes do sistema.
package interfaces

import (
	"context"
	"time"

	"github.com/google/uuid"

	"towercore/internal/core/domain"
)

// SiteEnvironmentRepository define a interface para repositórios de medições
// de ambiente e condições físicas do site.
type SiteEnvironmentRepository interface {
	// Create insere uma nova medição de ambiente
	Create(ctx context.Context, env *domain.SiteEnvironment) error
	// CreateMany insere múltiplas medições de uma vez (ótimo para batch)
	CreateMany(ctx context.Context, envs []*domain.SiteEnvironment) error
	// List retorna medições baseado em filtros
	List(ctx context.Context, filter *domain.SiteEnvironmentFilter) ([]*domain.SiteEnvironment, int, error)
	// GetLatest retorna a medição mais recente para um site
	GetLatest(ctx context.Context, siteID uuid.UUID) (*domain.SiteEnvironment, error)
	// DeleteOlderThan remove medições mais antigas que um determinado tempo (para retenção)
	DeleteOlderThan(ctx context.Context, olderThan time.Time) error
}