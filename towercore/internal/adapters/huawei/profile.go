package huawei

import (
	"towercore/internal/adapters/snmp"
	"towercore/internal/core/domain"
)

// Profile baseado no IMAP_NORTHBOUND_MIB-V2
func Profile() snmp.Profile {
	return snmp.Profile{
		Vendor: "huawei",

		Metrics: []snmp.MetricDefinition{
			// Heartbeat
			{
				OID:   ".1.3.6.1.4.1.2011.2.15.2.1.2.1.1.1.2",
				Key:   "heartbeat_period_s",
				Scale: 1,
			},
			{
				OID:   ".1.3.6.1.4.1.2011.2.15.2.1.2.1.1.1.3",
				Key:   "heartbeat_timestamp",
				Scale: 1,
			},

			// Configurable heartbeat service
			{
				OID:   ".1.3.6.1.4.1.2011.2.15.2.1.3.1.1",
				Key:   "heartbeat_report_interval_s",
				Scale: 1,
			},

			// Alarm query trigger
			{
				OID:   ".1.3.6.1.4.1.2011.2.15.2.4.1.5",
				Key:   "alarm_query_state",
				Scale: 1,
			},

			// Alarm metadata (latest/current trap context)
			{
				OID:   ".1.3.6.1.4.1.2011.2.15.2.4.3.3.11",
				Key:   "alarm_level",
				Scale: 1,
			},
			{
				OID:   ".1.3.6.1.4.1.2011.2.15.2.4.3.3.12",
				Key:   "alarm_restore_state",
				Scale: 1,
			},
			{
				OID:   ".1.3.6.1.4.1.2011.2.15.2.4.3.3.13",
				Key:   "alarm_confirm_state",
				Scale: 1,
			},
			{
				OID:   ".1.3.6.1.4.1.2011.2.15.2.4.3.3.50",
				Key:   "alarm_service_affect",
				Scale: 1,
			},
		},

		Alarms: []snmp.AlarmRule{
			// heartbeat stopped
			{
				Key:       "heartbeat_period_s",
				Severity:  domain.EventSeverityCritical,
				Threshold: 3600,
				Condition: "gt",
				Message:   "heartbeat timeout",
			},

			// invalid heartbeat
			{
				Key:       "heartbeat_period_s",
				Severity:  domain.EventSeverityWarning,
				Threshold: 3,
				Condition: "lt",
				Message:   "heartbeat interval below minimum",
			},

			// high severity alarm from NE
			{
				Key:       "alarm_level",
				Severity:  domain.EventSeverityCritical,
				Threshold: 5,
				Condition: "eq",
				Message:   "critical network alarm received",
			},

			// service affecting alarm
			{
				Key:       "alarm_service_affect",
				Severity:  domain.EventSeverityCritical,
				Threshold: 1,
				Condition: "eq",
				Message:   "service affecting alarm",
			},
		},
	}
}
