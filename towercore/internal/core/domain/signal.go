package domain

import "time"

type Signal struct {
	ID string
	DeviceID string
	SignalID int
	Name string
	Key string
	Value string
	Unit string
	Precision int
	DisplayTime time.Time
	UpdatedAt time.Time
}