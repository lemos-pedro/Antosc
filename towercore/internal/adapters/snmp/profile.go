package snmp

import "towercore/internal/core/domain"

type MetricDefinition struct {
	OID   string
	Key   string
	Scale float64
}

type AlarmRule struct {
	Key       string
	Severity  domain.EventSeverity
	Threshold float64
	Condition string
	Message   string
}

type Profile struct {
	Vendor  string
	Metrics []MetricDefinition
	Alarms  []AlarmRule
}
