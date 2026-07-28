package domain

import "time"

// TowerStatus representa o estado operacional de uma torre.
type TowerStatus string

type CollectionStatus string

const (
	TowerStatusOnline   TowerStatus = "online"
	TowerStatusDegraded TowerStatus = "degraded"
	TowerStatusOffline  TowerStatus = "offline"
	TowerStatusNoData   TowerStatus = "no_data"

	CollectionStatusNotConfigured  CollectionStatus = "not_configured"
	CollectionStatusNoCredentials  CollectionStatus = "no_credentials"
	CollectionStatusNeverCollected CollectionStatus = "never_collected"
	CollectionStatusFailed         CollectionStatus = "collection_failed"
	CollectionStatusActive         CollectionStatus = "active"
)

// Tower é a entidade central do domínio.
type Tower struct {
	ID                  string           `json:"tower_id"`
	Name                string           `json:"name"`
	Status              TowerStatus      `json:"status"`
	CollectionStatus    CollectionStatus `json:"collection_status"`
	LastCollectedAt     *time.Time       `json:"last_collected_at,omitempty"`
	LastSuccessfulAt    *time.Time       `json:"last_successful_at,omitempty"`
	LastCollectionError string           `json:"last_collection_error,omitempty"`

	// Deprecated: use Operators. Mantido para retrocompatibilidade com o
	// contrato v0 da API e código legado enquanto a migração para
	// site_operators (N:N) não estiver completa em toda a stack.
	OperatorID string `json:"operator_id,omitempty"`

	// Operators lista todos os operadores com equipamento neste site
	// (relação N:N via tabela site_operators). Um site pode ter mais de
	// um operador, cada um com o seu armário/equipamento próprio.
	Operators []Operator `json:"operators,omitempty"`

	RegionID       string `json:"region_id"`
	Vendor         string `json:"vendor,omitempty"`
	SNMPEnabled    bool   `json:"snmp_enabled,omitempty"`
	SNMPVersion    string `json:"snmp_version,omitempty"`
	SNMPTarget     string `json:"snmp_target,omitempty"`
	SNMPCommunity  string `json:"-"`
	SNMPV3User     string `json:"snmp_v3_user,omitempty"`
	SNMPAuthProto  string `json:"snmp_auth_protocol,omitempty"`
	SNMPAuthPass   string `json:"-"`
	SNMPPrivProto  string `json:"snmp_priv_protocol,omitempty"`
	SNMPPrivPass   string `json:"-"`
	NagiosEnabled  bool   `json:"nagios_enabled,omitempty"`
	NagiosHostname string `json:"nagios_hostname,omitempty"`
	NetecoEnabled  bool
	NetecoNEID     string
	NetecoSiteName string

	// Energia (via NetEco analyseJobs/siteCounterInfo)
	DCOutputVoltage  *float64 `json:"dc_output_voltage,omitempty"`
	DCLoadCurrent    *float64 `json:"dc_load_current,omitempty"`
	RectifierCurrent *float64 `json:"rectifier_current,omitempty"`

	// Bateria (via NetEco batteryManager/sohTreeRoaService)
	BatterySOC         *float64   `json:"battery_soc,omitempty"`
	BatterySOH         *float64   `json:"battery_soh,omitempty"`
	BatteryBackupTimeH *float64   `json:"battery_backup_time_h,omitempty"`
	BatteryUpdatedAt   *time.Time `json:"battery_updated_at,omitempty"`

	Availability30d float64   `json:"availability_30d,omitempty"`
	Availability7d  *float64  `json:"availability_7d,omitempty"`
	UpdatedAt       time.Time `json:"updated_at"`
	CreatedAt       time.Time `json:"created_at"`
}

// IsOperational retorna true se a torre está online ou degradada.
func (t *Tower) IsOperational() bool {
	return t.Status == TowerStatusOnline || t.Status == TowerStatusDegraded
}

// HasOperator retorna true se o operador dado tem equipamento nesta torre.
func (t *Tower) HasOperator(operatorID string) bool {
	for _, op := range t.Operators {
		if op.OperatorID == operatorID {
			return true
		}
	}
	return false
}
