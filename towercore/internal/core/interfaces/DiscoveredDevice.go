package interfaces

import (
	"context"
	"errors"

	"towercore/internal/core/domain"
)

var ErrDiscoveredDeviceNotFound = errors.New("discovered device not found")

// DiscoveredDeviceFilter define os filtros aceites na listagem de
// dispositivos descobertos.
type DiscoveredDeviceFilter struct {
	Status DiscoveredDeviceStatusFilter
	Limit  int
	Offset int
}

// DiscoveredDeviceStatusFilter evita repetir o tipo domain.DiscoveredDeviceStatus
// aqui e permite "sem filtro" (string vazia) de forma explícita.
type DiscoveredDeviceStatusFilter string

// DiscoveredDeviceRepository define o contrato de persistência para a
// staging table de network discovery.
type DiscoveredDeviceRepository interface {
	// List devolve os dispositivos descobertos, filtrados por status.
	List(ctx context.Context, filter DiscoveredDeviceFilter) ([]domain.DiscoveredDevice, int, error)

	// GetByID devolve um dispositivo descoberto específico.
	GetByID(ctx context.Context, id string) (*domain.DiscoveredDevice, error)

	// UpsertSeen regista (ou atualiza) que um IP respondeu a uma
	// sondagem agora mesmo — usado pelo discovery em cada ciclo de scan.
	// Se o IP já existir, atualiza apenas last_seen_at e os campos
	// detetados (sys_object_id, vendor), sem tocar em Status (não
	// reabre um dispositivo já 'promoted' ou 'ignored').
	UpsertSeen(ctx context.Context, device *domain.DiscoveredDevice) error

	// UpdateStatus marca um dispositivo como promovido ou ignorado.
	// Quando status é 'promoted', promotedTowerID deve ser o ID da
	// Tower criada a partir deste dispositivo.
	UpdateStatus(ctx context.Context, id string, status domain.DiscoveredDeviceStatus, promotedTowerID string) error
}
