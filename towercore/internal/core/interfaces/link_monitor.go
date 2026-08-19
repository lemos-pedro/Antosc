package interfaces

import (
	"context"

	"towercore/internal/core/domain"
)

// LinkMonitor é o port que qualquer adapter capaz de recolher métricas de
// uma interface de router deve implementar. A primeira implementação é
// adapters/snmp/router_transit (IF-MIB genérico via SNMP), mas o contrato
// fica aberto para futuras fontes (ex: API REST de um router gerido).
type LinkMonitor interface {
	// GetInterfaceMetrics faz a recolha SNMP pontual de uma interface
	// identificada por IP do router + ifIndex, devolvendo os contadores
	// brutos (ainda sem cálculo de bps/utilização — isso é responsabilidade
	// do core/services, que tem acesso ao snapshot anterior).
	GetInterfaceMetrics(ctx context.Context, routerIP string, ifIndex int) (*domain.LinkMetricSnapshot, error)

	// DiscoverInterfaces faz um SNMP walk a ifDescr/ifAlias/ifIndex no router
	// indicado, para permitir descoberta/reconciliação de novas interfaces
	// sem configuração manual (equivalente ao LLD do Zabbix).
	DiscoverInterfaces(ctx context.Context, routerIP string) ([]InterfaceDescriptor, error)
}

// InterfaceDescriptor é o resultado bruto de uma descoberta SNMP, antes de
// ser mapeado para domain.NetworkLink (mapeamento fica a cargo de quem
// consome DiscoverInterfaces, tipicamente uma rotina de onboarding manual
// ou semi-automática).
type InterfaceDescriptor struct {
	IfIndex int
	IfDescr string
	IfAlias string
	IfType  int
}
