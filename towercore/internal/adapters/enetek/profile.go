package enetek

import (
	"towercore/internal/adapters/snmp"
	"towercore/internal/core/domain"
)

// Profile para equipamentos Enetek Power System.
//
// Reconstruído a partir do MIB oficial (enetek_J.mib), enterprise OID
// real = enterprises.53318 (o profile anterior usava 1.3.6.1.4.1.99999,
// um placeholder que nunca existiu em nenhum equipamento real — causa
// provável do erro "samples is required" nas torres Enetek).
//
// Estrutura de base: .1.3.6.1.4.1.53318
//   acinfo            = enetek.1   -> acmainsmonitor1 (1) -> acmainsmonitor1int (1)
//   dcinfo            = enetek.2   -> dcmonitor (1) -> dcmonitorinfo (1)
//   rectifier         = enetek.3   -> rectifiers (1, tabela) / rectifiersinfo (2)
//   sysinfo           = enetek.100
//
// Todos os objetos usados aqui são escalares (não SEQUENCE OF), por isso
// levam sufixo ".0" no OID para SNMP GET — confirmado no MIB (não estão
// dentro de nenhum bloco "SYNTAX SEQUENCE OF ...Entry").
//
// VALIDADO POR SNMP WALK REAL (2026-07-09, site HBHUB004):
//   .1.1.1.1.0 = 2403  -> ×0.1 = 240.3V  (fase A, dentro do normal)
//   .1.1.1.2.0 = 2380  -> ×0.1 = 238.0V  (fase B, dentro do normal)
//   .1.1.1.3.0 = 0     -> ×0.1 = 0.0V    (fase C)
// Scale 0.1 confirmado correto para as 3 fases. A fase C lê exatamente 0
// de forma consistente — não é falha, é uma fase fisicamente não cablada
///não monitorizada neste site (equipamento aparentemente bi-fásico, não
// trifásico). Com a regra antiga (lt 200, sem distinguir "0 real" de
// "0 = não aplicável"), isto gerava um alarme permanente em warning —
// responsável pela maior fatia dos alarmes abertos na frota (27 de 51 no
// levantamento de 2026-07-09). Corrigido com IgnoreZero: true nas 3 fases.
func Profile() snmp.Profile {
	const base = ".1.3.6.1.4.1.53318"

	return snmp.Profile{
		Vendor: "enetek",

		Metrics: []snmp.MetricDefinition{
			// ======================
			// AC Mains (way 1)
			// acmainsmonitor1int = acinfo(1).acmainsmonitor1(1).int(1)
			//                    = enetek.1.1.1
			// ======================
			{
				OID:   base + ".1.1.1.1.0", // ac1voltagephaseaint, UNITS 0.1Volts
				Key:   "ac1_voltage_phase_a_v",
				Scale: 0.1, // validado por walk real: raw=2403 -> 240.3V
			},
			{
				OID:   base + ".1.1.1.2.0", // ac1voltagephasebint
				Key:   "ac1_voltage_phase_b_v",
				Scale: 0.1, // validado por walk real: raw=2380 -> 238.0V
			},
			{
				OID:   base + ".1.1.1.3.0", // ac1voltagephasecint
				Key:   "ac1_voltage_phase_c_v",
				Scale: 0.1, // fase possivelmente não cablada neste site — ver IgnoreZero abaixo
			},
			{
				// AINDA NÃO VALIDADO: unidade "0.1Amps" é assumida do MIB,
				// não confirmada por walk. Confirmar antes de alarmar em
				// cima disto.
				OID:   base + ".1.1.1.4.0", // ac1currentphaseaint, UNITS 0.1Amps (assumido, ver nota)
				Key:   "ac1_current_phase_a_a",
				Scale: 0.1,
			},
			{
				OID:   base + ".1.1.1.7.0", // ac1frequencyint
				Key:   "ac1_frequency_hz",
				Scale: 0.1,
			},

			// ======================
			// DC / Rectifier / Battery
			// dcmonitorinfo = dcinfo(2).dcmonitor(1).info(1) = enetek.2.1.1
			// ======================
			{
				OID:   base + ".2.1.1.1.0", // systembusvoltageint, UNITS 0.1Volts
				Key:   "system_bus_voltage_v",
				Scale: 0.1,
			},
			{
				OID:   base + ".2.1.1.2.0", // rectifiervoltageint, UNITS 0.1Volts
				Key:   "rectifier_voltage_v",
				Scale: 0.1,
			},
			{
				OID:   base + ".2.1.1.3.0", // rectifiercurrentint, UNITS 0.1Amps
				Key:   "rectifier_current_a",
				Scale: 0.1,
			},
			{
				OID:   base + ".2.1.1.4.0", // loadcurrentint, UNITS 0.1Amps
				Key:   "load_current_a",
				Scale: 0.1,
			},
			{
				OID:   base + ".2.1.1.5.0", // systempowerint, UNITS 0.01KW
				Key:   "system_power_kw",
				Scale: 0.01,
			},
			{
				OID:   base + ".2.1.1.6.0", // batterycurrentint, UNITS 0.1Amps
				Key:   "battery_current_a",
				Scale: 0.1,
			},
			{
				OID:   base + ".2.1.1.7.0", // batteryworkstate: 0=float,1=equalizing,2=test,3=discharge
				Key:   "battery_work_state",
				Scale: 1,
			},
			{
				OID:   base + ".2.1.1.8.0", // batterytemperatureint, UNITS 0.1 DegC
				Key:   "battery_temperature_c",
				Scale: 0.1,
			},
			{
				OID:   base + ".2.1.1.9.0", // ambienttemperatureint, UNITS 0.1 DegC
				Key:   "ambient_temperature_c",
				Scale: 0.1,
			},
			{
				OID:   base + ".2.1.1.10.0", // batterysurpluscapacityint, UNITS AH
				Key:   "battery_remaining_ah",
				Scale: 1,
			},

			// ======================
			// System info
			// sysinfo = enetek.100
			// ======================
			{
				OID:   base + ".100.7.0", // systemDateTime
				Key:   "system_datetime_raw",
				Scale: 1,
			},
		},

		// ASSUNÇÃO AINDA NÃO RESOLVIDA: thresholds de bateria (temperatura,
		// capacidade) continuam a espelhar os do Eltek, por falta de
		// validação de campo própria para Enetek. Já sabemos, pelo caso
		// Eltek, que copiar thresholds sem confirmar contra a realidade da
		// frota gera floods — isto é uma dívida técnica conhecida, não uma
		// correção definitiva. Se aparecerem muitos alarmes de
		// battery_temperature_c/battery_remaining_ah em torres Enetek,
		// tratar com a mesma prioridade que o caso mains aqui.
		Alarms: []snmp.AlarmRule{
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

			// ======================
			// Mains AC — 3 fases
			// CORRIGIDO: IgnoreZero adicionado nas 3. Validado por walk
			// real que 0V é "fase não cablada/monitorizada" neste modelo/
			// site, não falha real — sem isto, sites bi-fásicos disparavam
			// warning permanente na fase ausente (27 dos 51 alarmes
			// abertos na frota, no levantamento de 2026-07-09).
			//
			// ADICIONADO: tier crítico, que não existia — uma fase
			// presente mas em falha total ainda só disparava warning.
			// Threshold crítico (100V) é uma estimativa de engenharia
			// (abaixo do qual a rede está claramente inoperante para
			// equipamento de telecom), NÃO validada em campo. Ajustar
			// se o Director de O&M tiver um valor acordado diferente.
			// ======================
			{
				Key:        "ac1_voltage_phase_a_v",
				Severity:   domain.EventSeverityWarning,
				Threshold:  200,
				Condition:  "lt",
				Message:    "mains phase A low voltage",
				IgnoreZero: true,
			},
			{
				Key:        "ac1_voltage_phase_a_v",
				Severity:   domain.EventSeverityCritical,
				Threshold:  100,
				Condition:  "lt",
				Message:    "mains phase A failure",
				IgnoreZero: true,
			},
			{
				Key:        "ac1_voltage_phase_b_v",
				Severity:   domain.EventSeverityWarning,
				Threshold:  200,
				Condition:  "lt",
				Message:    "mains phase B low voltage",
				IgnoreZero: true,
			},
			{
				Key:        "ac1_voltage_phase_b_v",
				Severity:   domain.EventSeverityCritical,
				Threshold:  100,
				Condition:  "lt",
				Message:    "mains phase B failure",
				IgnoreZero: true,
			},
			{
				Key:        "ac1_voltage_phase_c_v",
				Severity:   domain.EventSeverityWarning,
				Threshold:  200,
				Condition:  "lt",
				Message:    "mains phase C low voltage",
				IgnoreZero: true,
			},
			{
				Key:        "ac1_voltage_phase_c_v",
				Severity:   domain.EventSeverityCritical,
				Threshold:  100,
				Condition:  "lt",
				Message:    "mains phase C failure",
				IgnoreZero: true,
			},
		},
	}
}
