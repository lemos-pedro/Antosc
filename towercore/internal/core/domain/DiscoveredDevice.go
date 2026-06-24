package domain

import "time"

// DiscoveredDeviceStatus representa o estado de triagem de um dispositivo
// encontrado por network discovery SNMP.
type DiscoveredDeviceStatus string

const (
	DiscoveredDeviceStatusPending  DiscoveredDeviceStatus = "pending"
	DiscoveredDeviceStatusPromoted DiscoveredDeviceStatus = "promoted"
	DiscoveredDeviceStatusIgnored  DiscoveredDeviceStatus = "ignored"
)

// DiscoveredDevice é um IP que respondeu a uma sondagem SNMP durante o
// network discovery, mas que ainda não tem os dados (nome, operador,
// região) necessários para se tornar uma Tower. Fica em triagem até
// um operador humano promover (criando uma Tower via TowerService) ou
// ignorar a entrada.
type DiscoveredDevice struct {
	ID              string                 `json:"device_id"`
	IPAddress       string                 `json:"ip_address"`
	SysObjectID     string                 `json:"sys_object_id,omitempty"`
	DetectedVendor  string                 `json:"detected_vendor,omitempty"`
	SNMPVersion     string                 `json:"snmp_version,omitempty"`
	SNMPCommunity   string                 `json:"-"`
	Status          DiscoveredDeviceStatus `json:"status"`
	PromotedTowerID string                 `json:"promoted_tower_id,omitempty"`
	FirstSeenAt     time.Time              `json:"first_seen_at"`
	LastSeenAt      time.Time              `json:"last_seen_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	CreatedAt       time.Time              `json:"created_at"`
}
