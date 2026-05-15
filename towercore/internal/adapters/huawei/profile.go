package huawei

import (
	"towercore/internal/adapters/snmp"
	"towercore/internal/core/domain"
)

// Profile baseado no IMAP_NORTHBOUND_MIB-V2.
func Profile() snmp.Profile {
	return snmp.Profile{
		Vendor: "huawei",
		Metrics: []snmp.MetricDefinition{
			{OID: ".1.3.6.1.4.1.2011.2.15.2.1.2.1.1.1.2", Key: "heartbeat_period_s", Scale: 1},
		},
		Alarms: []snmp.AlarmRule{
			{Key: "heartbeat_period_s", Severity: domain.EventSeverityWarning, Threshold: 3600, Condition: "gt", Message: "heartbeat period above expected range"},
			{Key: "heartbeat_period_s", Severity: domain.EventSeverityWarning, Threshold: 3, Condition: "lt", Message: "heartbeat period below expected range"},
		},
	}
}
