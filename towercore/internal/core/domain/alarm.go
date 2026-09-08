package domain

import "time"

type AlarmSeverity string

const (
	AlarmInfo     AlarmSeverity = "info"
	AlarmWarning  AlarmSeverity = "warning"
	AlarmMajor    AlarmSeverity = "major"
	AlarmCritical AlarmSeverity = "critical"
)

type DeviceAlarm struct {
	ID        string
	DeviceID  string
	AlarmID   int
	Name      string
	Message   string
	Severity  AlarmSeverity
	Active    bool
	StartedAt time.Time
	EndedAt   *time.Time
}
