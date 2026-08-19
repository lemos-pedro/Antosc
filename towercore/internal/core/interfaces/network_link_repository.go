package interfaces

import (
	"context"

	"towercore/internal/core/domain"
)

// NetworkLinkRepository é o port de persistência para links de rede.
// Implementado em infrastructure/database, seguindo o mesmo padrão dos
// repositórios já existentes para Tower.
type NetworkLinkRepository interface {
	Create(ctx context.Context, link *domain.NetworkLink) error
	Update(ctx context.Context, link *domain.NetworkLink) error
	GetByID(ctx context.Context, linkID string) (*domain.NetworkLink, error)
	GetByRouterAndIfIndex(ctx context.Context, routerIP string, ifIndex int) (*domain.NetworkLink, error)
	List(ctx context.Context, limit, offset int) ([]*domain.NetworkLink, int, error)

	// SaveMetricSnapshot persiste uma leitura pontual (para série histórica
	// de throughput/erros). Implementação decide se guarda em tabela dedicada
	// ou agregada por intervalo.
	SaveMetricSnapshot(ctx context.Context, snapshot *domain.LinkMetricSnapshot) error

	// GetLastSnapshot devolve a última leitura conhecida, necessária para
	// calcular delta de contadores (bps) entre polls consecutivos.
	GetLastSnapshot(ctx context.Context, linkID string) (*domain.LinkMetricSnapshot, error)
}

// NetworkLinkEventRepository é o port de persistência para eventos/alarmes
// de link, seguindo o mesmo padrão CreateOrTouch/Resolve já usado para
// alarmes de torre (deduplicação via alarm_key).
type NetworkLinkEventRepository interface {
	CreateOrTouch(ctx context.Context, event *domain.LinkEvent) error
	Resolve(ctx context.Context, alarmKey string) error
	ListByLinkID(ctx context.Context, linkID string, limit, offset int) ([]*domain.LinkEvent, int, error)
}
