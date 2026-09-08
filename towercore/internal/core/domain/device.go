package domain

import "time"

type DeviceStatus string

const (
	DeviceStatusUnknown      DeviceStatus = "unknown"
	DeviceStatusConnected    DeviceStatus = "connected"
	DeviceStatusDisconnected DeviceStatus = "disconnected"
)

type Device struct {
	ID        string
	TowerID   string
	DN        string
	ParentDN  string
	Name      string
	MOCID     int
	TypeID    int
	TypeName  string
	Vendor    string
	Status    DeviceStatus
	LastSeen  time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}
