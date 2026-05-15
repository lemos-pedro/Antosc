package domain

import "time"

// TowerStatus representa o estado operacional de uma torre.
type TowerStatus string

const (
	TowerStatusOnline   TowerStatus = "online"
	TowerStatusDegraded TowerStatus = "degraded"
	TowerStatusOffline  TowerStatus = "offline"
)

// Tower é a entidade central do domínio.
type Tower struct {
	ID              string      `json:"tower_id"`
	Name            string      `json:"name"`
	Status          TowerStatus `json:"status"`
	OperatorID      string      `json:"operator_id"`
	RegionID        string      `json:"region_id"`
	Vendor          string      `json:"vendor,omitempty"`
	SNMPEnabled     bool        `json:"snmp_enabled,omitempty"`
	SNMPVersion     string      `json:"snmp_version,omitempty"`
	SNMPTarget      string      `json:"snmp_target,omitempty"`
	SNMPCommunity   string      `json:"-"`
	SNMPV3User      string      `json:"snmp_v3_user,omitempty"`
	SNMPAuthProto   string      `json:"snmp_auth_protocol,omitempty"`
	SNMPAuthPass    string      `json:"-"`
	SNMPPrivProto   string      `json:"snmp_priv_protocol,omitempty"`
	SNMPPrivPass    string      `json:"-"`
	Availability30d float64     `json:"availability_30d,omitempty"`
	UpdatedAt       time.Time   `json:"updated_at"`
	CreatedAt       time.Time   `json:"created_at"`
}

// IsOperational retorna true se a torre está online ou degradada.
func (t *Tower) IsOperational() bool {
	return t.Status == TowerStatusOnline || t.Status == TowerStatusDegraded
}
