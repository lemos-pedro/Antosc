package snmp

import (
	"context"

	"towercore/internal/core/domain"
)

// Collector define o contrato de coleta de metricas SNMP para uma torre,
// dado um perfil de equipamento (vendor/OIDs). E implementado por
// GoSNMPCollector (producao) e pode ser implementado por mocks/fakes em
// testes ou simulacao (ex.: snmpsim).
type Collector interface {
	Collect(ctx context.Context, tower domain.Tower, profile Profile) (map[string]float64, error)
}
