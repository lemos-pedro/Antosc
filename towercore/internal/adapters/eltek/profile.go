package eltek

import (
	"towercore/internal/adapters/snmp"
	"towercore/internal/core/domain"
)

// Production profile para Eltek Flatpack2 / SP2 Touch
//
// Validado por SNMP walk real contra site LDVIA015 (192.168.203.5,
// 2026-07-09). NÃO assumir a configuração do template Zabbix como
// definitiva — esse template define só 2 retificadores e mains L1/L2;
// este hardware real tem 3 retificadores FLATPACK2 48/3000 HE e as 3
// fases de mains. Variantes de configuração existem na frota; o número
// de retificadores/fases por site deve ser confirmado por walk antes de
// assumir cobertura completa em qualquer torre nova.
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
				Scale: 0.01, // confirmado: MULTIPLIER 0.01 no template
			},
			{
				OID:   ".1.3.6.1.4.1.12148.10.10.6.5.0",
				Key:   "battery_current_a",
				Scale: 0.1, // confirmado: MULTIPLIER 0.1 no template
			},
			{
				OID:          ".1.3.6.1.4.1.12148.10.10.7.5.0",
				Key:          "battery_temperature_c",
				Scale:        1,               // confirmado: sem multiplicador no template
				IgnoreValues: []float64{-100}, // slot/sensor vazio, não é temperatura real
			},
			{
				// No walk de CAOTINHA, .12.2.0 é apenas o rótulo
				// "BatteryQuality"; .12.5.0 é o valor. Zero significa que
				// ainda não houve ciclo de teste, nunca uma falha de bateria.
				OID:                ".1.3.6.1.4.1.12148.10.10.12.5.0",
				Key:                "battery_quality",
				Scale:              1,
				ZeroMeansNotTested: true,
			},
			{
				OID:   ".1.3.6.1.4.1.12148.10.10.9.5.0",
				Key:   "battery_remaining_",
				Scale: 1,
			},
			{
				OID:   ".1.3.6.1.4.1.12148.10.10.11.5.0",
				Key:   "battery_total_",
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
				Scale: 1, // confirmado: MULTIPLIER 0.1 no template
			},
			{
				OID:   ".1.3.6.1.4.1.12148.10.9.9.1.6.1.1",
				Key:   "load_voltage_v",
				Scale: 0.01, // confirmado: MULTIPLIER 0.01 no template
			},
			{
				OID:   ".1.3.6.1.4.1.12148.10.9.1.0",
				Key:   "load_status",
				Scale: 1,
			},

			// ======================
			// Mains
			// ======================
			{
				OID:   ".1.3.6.1.4.1.12148.10.3.1.0",
				Key:   "mains_status",
				Scale: 1,
			},
			{
				// CORRIGIDO: Scale era 10, agora 1 (validado por SNMP walk
				// real: raw=219 em CTR5ABRIL1/192.168.203.5, consistente
				// com tensão AC real ~220V).
				OID:   ".1.3.6.1.4.1.12148.10.3.4.1.6.1",
				Key:   "mains_voltage_l1_v",
				Scale: 1,
			},
			{
				// CORRIGIDO: Scale era 10, agora 1 (validado: raw=224).
				OID:   ".1.3.6.1.4.1.12148.10.3.4.1.6.2",
				Key:   "mains_voltage_l2_v",
				Scale: 1,
			},
			{
				// REINTEGRADO em 2026-07-09: tinha sido removido por
				// suposição (ausente no template Zabbix de referência).
				// SNMP walk real a 192.168.203.5 devolveu raw=222 neste
				// índice — o OID EXISTE e é válido neste hardware. O
				// template Zabbix aparentemente cobre uma variante de
				// 2 fases; este equipamento tem 3 fases reais.
				// Lição: a ausência de um item no template não prova que
				// o OID não existe no dispositivo — só o walk prova.
				OID:   ".1.3.6.1.4.1.12148.10.3.4.1.6.3",
				Key:   "mains_voltage_l3_v",
				Scale: 1,
			},

			// ======================
			// Rectifiers
			// ======================
			{
				// CORRIGIDO: Scale era 0.1, template não tem multiplicador.
				// Validado por walk real: raw=218 (~igual a mains AC).
				OID:   ".1.3.6.1.4.1.12148.10.5.6.1.4.1",
				Key:   "rectifier_1_input_v",
				Scale: 1,
			},
			{
				// CORRIGIDO: Scale era 0.1, template não tem multiplicador.
				OID:   ".1.3.6.1.4.1.12148.10.5.6.1.4.2",
				Key:   "rectifier_2_input_v",
				Scale: 1,
			},
			{
				// ADICIONADO: SNMP walk real (site LDVIA015/192.168.203.5)
				// mostrou 3 retificadores FLATPACK2 48/3000 HE com números
				// de série próprios (.10.5.6.1.10.1/2/3 = 231950075366 /
				// 231950075364 / 231950075317) — o profile só cobria os
				// índices 1 e 2. Sem isto, falha do 3º módulo é invisível.
				OID:   ".1.3.6.1.4.1.12148.10.5.6.1.4.3",
				Key:   "rectifier_3_input_v",
				Scale: 1,
			},
			{
				OID:   ".1.3.6.1.4.1.12148.10.5.6.1.3.1",
				Key:   "rectifier_1_output_a",
				Scale: 0.1, // confirmado: MULTIPLIER 0.1 no template
			},
			{
				OID:   ".1.3.6.1.4.1.12148.10.5.6.1.3.2",
				Key:   "rectifier_2_output_a",
				Scale: 0.1, // confirmado: MULTIPLIER 0.1 no template
			},
			{
				// ADICIONADO: ver nota acima sobre o retificador 3.
				OID:   ".1.3.6.1.4.1.12148.10.5.6.1.3.3",
				Key:   "rectifier_3_output_a",
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
				// ADICIONADO: ver nota acima sobre o retificador 3.
				OID:   ".1.3.6.1.4.1.12148.10.5.6.1.2.3",
				Key:   "rectifier_3_status",
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

			// ======================
			// Battery temperature
			// ATENÇÃO: o template Zabbix usa $BATTERY_TEMP_HIGH_WARNING=25,
			// mas esse valor foi copiado sem validação de campo. Em clima
			// tropical (Angola), 25°C é temperatura ambiente normal — a
			// CAOTINHA reporta 31°C de forma consistente sem sinal de
			// degradação real. Aplicar 25 como warning, combinado com a
			// regra "hasWarning => Degraded" no SNMPIngestService, marca a
			// frota inteira como degradada.
			// Mantido o valor original do Antosc (40) como warning até
			// haver validação empírica de uma baseline real de campo
			// (média de battery_temperature_c em condições normais numa
			// amostra de torres, em vários horários do dia).
			// ======================
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
			// Alarmes de baixa temperatura do template mantidos: risco real
			// mesmo em clima tropical (madrugadas em altitude, falha de
			// climatização do shelter), e -20/0°C está longe da baseline
			// normal — sem o mesmo risco de flood.
			{
				Key:       "battery_temperature_c",
				Severity:  domain.EventSeverityWarning,
				Threshold: 0,
				Condition: "lt",
				Message:   "battery temperature low",
			},
			{
				Key:       "battery_temperature_c",
				Severity:  domain.EventSeverityCritical,
				Threshold: -20,
				Condition: "lt",
				Message:   "battery temperature critically low",
			},

			// ======================
			// Battery capacity
			// CORRIGIDO: o warning absoluto de 20  fixo é o que está a
			// inundar a frota de "degradada" — bancos de bateria pequenos
			// passam a maior parte da vida útil normal abaixo de 20
			// restantes, sem que isso signifique falha real.
			// battery_total_ já é coletado — o correto é avaliar como
			// percentagem (battery_remaining_ / battery_total_), não
			// como valor absoluto. Requer suporte a regra derivada no
			// motor de avaliação (matchCondition); até lá, manter apenas
			// o crítico absoluto de 5  (bateria quase esgotada em
			// qualquer capacidade de banco) e remover o warning de 20 .
			// ======================
			{
				Key:       "battery_remaining_",
				Severity:  domain.EventSeverityCritical,
				Threshold: 5,
				Condition: "lt",
				Message:   "battery critically low",
			},
			// TODO: reintroduzir warning como regra percentual, ex.:
			// (battery_remaining_ / battery_total_) < 0.20
			// assim que o motor de alarmes suportar métricas derivadas.

			// ======================
			// Mains FAILURE
			// Igual ao Zabbix: tensão AC abaixo de 80V.
			// Regra de L3 REMOVIDA — ver nota nas Metrics acima.
			// ======================
			{
				Key:        "mains_voltage_l1_v",
				Severity:   domain.EventSeverityWarning,
				Threshold:  80,
				Condition:  "lt",
				Message:    "mains power failure L1",
				IgnoreZero: true,
			},
			{
				Key:        "mains_voltage_l2_v",
				Severity:   domain.EventSeverityWarning,
				Threshold:  80,
				Condition:  "lt",
				Message:    "mains power failure L2",
				IgnoreZero: true,
			},
			{
				Key:        "mains_voltage_l3_v",
				Severity:   domain.EventSeverityWarning,
				Threshold:  80,
				Condition:  "lt",
				Message:    "mains power failure L3",
				IgnoreZero: true,
			},

			// Status AC apenas se Eltek reportar erro real
			{
				Key:       "mains_status",
				Severity:  domain.EventSeverityCritical,
				Threshold: 0,
				Condition: "eq",
				Message:   "mains status error",
			},

			// Rectifiers
			{
				Key:       "rectifiers_temperature_c",
				Severity:  domain.EventSeverityWarning,
				Threshold: 60,
				Condition: "gt",
				Message:   "rectifier temperature high",
			},

			// Load
			{
				Key:       "load_current_a",
				Severity:  domain.EventSeverityWarning,
				Threshold: 220, // anteriormente 180
				Condition: "gt",
				Message:   "load current high",
			},
		},
	}
}
