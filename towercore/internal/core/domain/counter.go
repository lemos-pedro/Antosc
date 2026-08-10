package domain

import "time"

type Counter struct {
	ID string
	DeviceID string
	CounterID int
	Name string
	Value float64
	Unit string
	Precision int
	Timestamp time.Time
}