package interfaces

import "context"

// TowerEndpoint é a leitura de um endpoint de equipamento por torre
// (ver migration 00006_tower_endpoints.sql). Fica em core/interfaces em vez
// de core/domain por agora, para reduzir superfície de mudança — pode
// mover-se para domain.TowerEndpoint mais tarde se ganhar mais campos/regras.
type TowerEndpoint struct {
	ID            string
	TowerID       string
	EquipmentType string // "rectifier", "generator", ...
	Protocol      string // "snmp", "modbus_tcp", "modbus_rtu"
	IPAddress     string
	Port          int
	SlaveID       int
	Credential    string // community string SNMP / credencial Modbus
	Enabled       bool
}

// TowerEndpointFilter segue o mesmo padrão de TowerFilter/EventFilter já
// usado nos outros repositórios (Limit/Offset com defaults geridos no
// service ou scheduler chamador).
type TowerEndpointFilter struct {
	TowerID       string
	EquipmentType string
	Enabled       *bool
	Limit         int
	Offset        int
}

// TowerEndpointRepository é o port para leitura/escrita de tower_endpoints.
type TowerEndpointRepository interface {
	List(ctx context.Context, filter TowerEndpointFilter) ([]TowerEndpoint, int, error)
	GetByTowerAndType(ctx context.Context, towerID, equipmentType string) (*TowerEndpoint, error)
	Create(ctx context.Context, ep *TowerEndpoint) error
	Update(ctx context.Context, ep *TowerEndpoint) error
}
