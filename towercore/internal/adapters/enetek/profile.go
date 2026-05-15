package enetek

import (
	"towercore/internal/adapters/snmp"
	"towercore/internal/core/domain"
)

// Profile inicial baseado na árvore de objetos do enetek_J.mib.
// OIDs de tabela usam índice 1 como ponto de partida.
func Profile() snmp.Profile {
	return snmp.Profile{
		Vendor: "enetek",
		Metrics: []snmp.MetricDefinition{
			{OID: ".1.3.6.1.4.1.53318.4.10.1.1.2.1", Key: "battery_voltage_v", Scale: 1},
			{OID: ".1.3.6.1.4.1.53318.4.10.1.1.3.1", Key: "battery_current_a", Scale: 1},
			{OID: ".1.3.6.1.4.1.53318.4.10.1.1.5.1", Key: "battery_temperature_c", Scale: 1},
			{OID: ".1.3.6.1.4.1.53318.4.10.1.1.4.1", Key: "battery_remaining_pct", Scale: 1},
		},
		Alarms: []snmp.AlarmRule{
			{Key: "battery_temperature_c", Severity: domain.EventSeverityWarning, Threshold: 45, Condition: "gt", Message: "battery temperature high"},
			{Key: "battery_remaining_pct", Severity: domain.EventSeverityCritical, Threshold: 20, Condition: "lt", Message: "battery remaining percentage low"},
		},
	}
}
