package eltek

import (
	"towercore/internal/adapters/snmp"
	"towercore/internal/core/domain"
)

// Profile baseado em OIDs presentes no template Eltek Flatpack S.
func Profile() snmp.Profile {
	return snmp.Profile{
		Vendor: "eltek",
		Metrics: []snmp.MetricDefinition{
			{OID: ".1.3.6.1.4.1.12148.10.10.5.5.0", Key: "battery_voltage_v", Scale: 0.01},
			{OID: ".1.3.6.1.4.1.12148.10.10.6.5.0", Key: "battery_current_a", Scale: 0.1},
			{OID: ".1.3.6.1.4.1.12148.10.10.7.5.0", Key: "battery_temperature_c", Scale: 1},
			{OID: ".1.3.6.1.4.1.12148.10.9.2.5.0", Key: "load_current_a", Scale: 0.1},
			{OID: ".1.3.6.1.4.1.12148.10.10.9.5.0", Key: "battery_remaining_ah", Scale: 1},
		},
		Alarms: []snmp.AlarmRule{
			{Key: "battery_temperature_c", Severity: domain.EventSeverityWarning, Threshold: 45, Condition: "gt", Message: "battery temperature high"},
			{Key: "battery_temperature_c", Severity: domain.EventSeverityCritical, Threshold: 50, Condition: "gt", Message: "battery temperature critical"},
			{Key: "battery_remaining_ah", Severity: domain.EventSeverityWarning, Threshold: 20, Condition: "lt", Message: "battery remaining capacity low"},
		},
	}
}
