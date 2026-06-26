package eltek

import (
	"towercore/internal/adapters/snmp"
	"towercore/internal/core/domain"
)

// Production profile para Eltek Flatpack2 / SP2 Touch
func Profile() snmp.Profile {
	return snmp.Profile{
		Vendor: "eltek",

		Metrics: []snmp.MetricDefinition{
			// ======================
			// Battery
			// ======================
			{
				OID:   ".1.3.6.1.4.1.12148.10.10.5.5.0",
				Key:   "battery_voltage_v",
				Scale: 0.01,
			},
			{
				OID:   ".1.3.6.1.4.1.12148.10.10.6.5.0",
				Key:   "battery_current_a",
				Scale: 0.1,
			},
			{
				OID:   ".1.3.6.1.4.1.12148.10.10.7.5.0",
				Key:   "battery_temperature_c",
				Scale: 1,
			},
			{
				OID:   ".1.3.6.1.4.1.12148.10.10.9.5.0",
				Key:   "battery_remaining_ah",
				Scale: 1,
			},
			{
				OID:   ".1.3.6.1.4.1.12148.10.10.11.5.0",
				Key:   "battery_total_ah",
				Scale: 1,
			},
			{
				OID:   ".1.3.6.1.4.1.12148.10.10.1.0",
				Key:   "battery_status",
				Scale: 1,
			},

			// ======================
			// Load
			// ======================
			{
				OID:   ".1.3.6.1.4.1.12148.10.9.2.5.0",
				Key:   "load_current_a",
				Scale: 0.1,
			},
			{
				OID:   ".1.3.6.1.4.1.12148.10.9.9.1.6.1.1",
				Key:   "load_voltage_v",
				Scale: 0.01,
			},
			{
				OID:   ".1.3.6.1.4.1.12148.10.9.1.0",
				Key:   "load_status",
				Scale: 1,
			},

			// ======================
			// Mains (3-phase)
			// ======================
			{
				OID:   ".1.3.6.1.4.1.12148.10.3.1.0",
				Key:   "mains_status",
				Scale: 1,
			},
			{
				OID:   ".1.3.6.1.4.1.12148.10.3.4.1.6.1",
				Key:   "mains_voltage_l1_v",
				Scale: 0.1,
			},
			{
				OID:   ".1.3.6.1.4.1.12148.10.3.4.1.6.2",
				Key:   "mains_voltage_l2_v",
				Scale: 0.1,
			},
			{
				OID:   ".1.3.6.1.4.1.12148.10.3.4.1.6.3",
				Key:   "mains_voltage_l3_v",
				Scale: 0.1,
			},

			// ======================
			// Rectifiers
			// ======================
			{
				OID:   ".1.3.6.1.4.1.12148.10.5.6.1.4.1",
				Key:   "rectifier_1_input_v",
				Scale: 0.1,
			},
			{
				OID:   ".1.3.6.1.4.1.12148.10.5.6.1.4.2",
				Key:   "rectifier_2_input_v",
				Scale: 0.1,
			},
			{
				OID:   ".1.3.6.1.4.1.12148.10.5.6.1.3.1",
				Key:   "rectifier_1_output_a",
				Scale: 0.1,
			},
			{
				OID:   ".1.3.6.1.4.1.12148.10.5.6.1.3.2",
				Key:   "rectifier_2_output_a",
				Scale: 0.1,
			},
			{
				OID:   ".1.3.6.1.4.1.12148.10.5.6.1.2.1",
				Key:   "rectifier_1_status",
				Scale: 1,
			},
			{
				OID:   ".1.3.6.1.4.1.12148.10.5.6.1.2.2",
				Key:   "rectifier_2_status",
				Scale: 1,
			},
			{
				OID:   ".1.3.6.1.4.1.12148.10.5.18.5.0",
				Key:   "rectifiers_temperature_c",
				Scale: 1,
			},

			// ======================
			// Controller
			// ======================
			{
				OID:   ".1.3.6.1.4.1.12148.10.13.11.2.1.6.1.1",
				Key:   "controller_temperature_c",
				Scale: 1,
			},

			// ======================
			// System
			// ======================
			{
				OID:   ".1.3.6.1.4.1.12148.10.2.1.0",
				Key:   "system_status",
				Scale: 1,
			},
		},

		Alarms: []snmp.AlarmRule{
			// Battery temperature
			{
				Key:       "battery_temperature_c",
				Severity:  domain.EventSeverityWarning,
				Threshold: 40,
				Condition: "gt",
				Message:   "battery temperature high",
			},
			{
				Key:       "battery_temperature_c",
				Severity:  domain.EventSeverityCritical,
				Threshold: 50,
				Condition: "gt",
				Message:   "battery temperature critical",
			},

			// Battery capacity
			{
				Key:       "battery_remaining_ah",
				Severity:  domain.EventSeverityWarning,
				Threshold: 20,
				Condition: "lt",
				Message:   "battery capacity low",
			},
			{
				Key:       "battery_remaining_ah",
				Severity:  domain.EventSeverityCritical,
				Threshold: 5,
				Condition: "lt",
				Message:   "battery critically low",
			},

			// Mains fail
			{
				Key:       "mains_status",
				Severity:  domain.EventSeverityCritical,
				Threshold: 1,
				Condition: "eq",
				Message:   "mains power failure",
			},

			// Undervoltage per phase
			{
				Key:       "mains_voltage_l1_v",
				Severity:  domain.EventSeverityWarning,
				Threshold: 200,
				Condition: "lt",
				Message:   "mains phase L1 low voltage",
			},
			{
				Key:       "mains_voltage_l2_v",
				Severity:  domain.EventSeverityWarning,
				Threshold: 200,
				Condition: "lt",
				Message:   "mains phase L2 low voltage",
			},
			{
				Key:       "mains_voltage_l3_v",
				Severity:  domain.EventSeverityWarning,
				Threshold: 200,
				Condition: "lt",
				Message:   "mains phase L3 low voltage",
			},

			// Rectifier overheat
			{
				Key:       "rectifiers_temperature_c",
				Severity:  domain.EventSeverityWarning,
				Threshold: 60,
				Condition: "gt",
				Message:   "rectifier temperature high",
			},

			// Load overload
			{
				Key:       "load_current_a",
				Severity:  domain.EventSeverityWarning,
				Threshold: 80,
				Condition: "gt",
				Message:   "load current high",
			},
		},
	}
}