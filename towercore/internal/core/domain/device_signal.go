package domain

import "time"

type DeviceSignal struct {

	ID int64

	DeviceDN string

	Name string

	Key string

	Value string

	Unit string

	CollectedAt time.Time
}