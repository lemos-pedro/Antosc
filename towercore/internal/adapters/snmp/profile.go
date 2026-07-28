package snmp

import "towercore/internal/core/domain"

type MetricDefinition struct {
	OID                string
	Key                string
	Scale              float64
	IgnoreValues       []float64
	ZeroMeansNotTested bool
}

type AlarmRule struct {
	Key        string
	Severity   domain.EventSeverity
	Threshold  float64
	Condition  string
	Message    string
	IgnoreZero bool // se true, valor exatamente 0 nunca dispara este alarme
	// (usado para métricas onde 0 significa "não aplicável/não
	// cablado", não "valor real de falha" — ex.: fase C de
	// mains num site só com 2 fases monitoradas)
}

type Profile struct {
	Vendor  string
	Metrics []MetricDefinition
	Alarms  []AlarmRule
}
