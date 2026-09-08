package vertiv

import (
	"towercore/internal/adapters/snmp"
	"towercore/internal/core/domain"
)

// Production profile para Vertiv Netsure NCU (Modelo M830B / NETSURE-012-C)
//
// Validado por SNMP walk real contra o site com controlador NCU
// (192.168.204.133, community "public" v2c, 2026-08-10). Enterprise OID
// real é .1.3.6.1.4.1.6302 (Vertiv/Emerson legacy) — NÃO 21239 como
// assumido inicialmente antes do walk. Confirma o princípio: nunca gerar
// OIDs a partir de suposição de fabricante, só do walk real.
//
// IMPORTANTE — nível de confiança dos campos abaixo:
//   - "CONFIRMADO" = valor bate com o que a própria UI web do controlador
//     mostrou no mesmo momento (Tensão de Saída 53.3V, Retificadores 4,
//     Todos os Alarmes(2)).
//   - "INFERIDO" = a posição/escala foi deduzida pela ordem da tabela SNMP
//     e por ser um valor plausível (ex.: tensão AC ~230V), mas NÃO foi
//     cruzado com um segundo ecrã da UI mostrando esse valor exato. Tratar
//     como candidato a confirmar num próximo ciclo de poll, comparando a
//     leitura ao vivo com a UI antes de usar para gerar alarmes reais.
//
// Não existe MIB pública oficial consultada para este mapeamento — é
// 100% baseado no walk. Se a Vertiv disponibilizar a NETSURE-MIB (ou
// MIB-6302) mais tarde, isso deve substituir/confirmar este mapeamento.
func Profile() snmp.Profile {
	return snmp.Profile{
		Vendor: "vertiv",

		Metrics: []snmp.MetricDefinition{

			// ======================
			// System / Output — CONFIRMADO contra UI
			// ======================
			{
				// raw=53202 no walk == 53.202V; UI mostrava "Tensão de
				// Saída 53.3V" no mesmo período. CONFIRMADO (scale 0.001).
				OID:   ".1.3.6.1.4.1.6302.2.1.2.2.0",
				Key:   "output_voltage_v",
				Scale: 0.001,
			},
			{
				// raw=51231 == 51.231V. Provável tensão de bateria
				// (ligeiramente abaixo da tensão flutuante de saída,
				// padrão esperado). INFERIDO — confirmar rótulo exato.
				OID:   ".1.3.6.1.4.1.6302.2.1.2.3.0",
				Key:   "battery_voltage_v",
				Scale: 0.001,
			},
			{
				// raw=19. Sem unidade clara no walk; candidato a
				// temperatura ambiente do NCU (19°C é plausível para
				// medição em shelter climatizado). INFERIDO, scale 1 —
				// confirmar contra ecrã "Inventário do Sistema" na UI.
				OID:   ".1.3.6.1.4.1.6302.2.1.2.4.0",
				Key:   "controller_ambient_temperature_c",
				Scale: 1,
			},

			// ======================
			// Battery — tabela indexada (só Battery 1 mapeada, index=116)
			// INFERIDO: raw=37908 pode ser corrente de bateria em mA.
			// Se houver mais de uma bateria fisicamente instalada, a
			// tabela .2.1.2.5.5.1.x.<index> tem uma linha por bateria —
			// este profile só cobre a primeira encontrada no walk.
			// ======================
			{
				OID:   ".1.3.6.1.4.1.6302.2.1.2.5.5.1.2.116",
				Key:   "battery_1_current_a",
				Scale: 0.001,
			},

			// ======================
			// Mains (AC) — INFERIDO por plausibilidade (~230V nominal),
			// mas NÃO cruzado com um ecrã da UI que mostre mains AC
			// diretamente (a UI só mostrava Saída DC). Confirmar antes de
			// usar em alarme de "mains failure".
			// ======================
			{
				OID:   ".1.3.6.1.4.1.6302.2.1.2.6.1.0",
				Key:   "mains_voltage_l1_v",
				Scale: 0.001,
			},
			{
				OID:   ".1.3.6.1.4.1.6302.2.1.2.6.2.0",
				Key:   "mains_voltage_l2_v",
				Scale: 0.001,
			},
			{
				OID:   ".1.3.6.1.4.1.6302.2.1.2.6.3.0",
				Key:   "mains_voltage_l3_v",
				Scale: 0.001,
			},

			// ======================
			// System temperature — tabela com 2 sensores no walk
			// (73778 = "System Temperature 1", 73779 = "System
			// Temperature 2"). INFERIDO scale 0.001 (33.4°C / 34.6°C
			// plausíveis para ambiente tropical).
			// ======================
			{
				OID:   ".1.3.6.1.4.1.6302.2.1.2.7.3.1.2.73778",
				Key:   "system_temperature_1_c",
				Scale: 0.001,
			},
			{
				OID:   ".1.3.6.1.4.1.6302.2.1.2.7.3.1.2.73779",
				Key:   "system_temperature_2_c",
				Scale: 0.001,
			},

			// ======================
			// Rectifiers — CONFIRMADO: count=4 bate com UI ("Retificadores 4")
			// ======================
			{
				OID:   ".1.3.6.1.4.1.6302.2.1.2.11.1.0",
				Key:   "rectifier_count",
				Scale: 1,
			},
			{
				// INFERIDO scale 0.001 — raw=22162 → 22.162A por
				// retificador; 4x ~22A é coerente com Corrente de Saída
				// 55A total da UI (soma real seria maior, pode incluir
				// perdas/consumo não distribuído — confirmar).
				OID:   ".1.3.6.1.4.1.6302.2.1.2.11.4.1.6.3",
				Key:   "rectifier_1_output_a",
				Scale: 0.001,
			},
			{
				OID:   ".1.3.6.1.4.1.6302.2.1.2.11.4.1.6.4",
				Key:   "rectifier_2_output_a",
				Scale: 0.001,
			},
			{
				OID:   ".1.3.6.1.4.1.6302.2.1.2.11.4.1.6.5",
				Key:   "rectifier_3_output_a",
				Scale: 0.001,
			},
			{
				OID:   ".1.3.6.1.4.1.6302.2.1.2.11.4.1.6.6",
				Key:   "rectifier_4_output_a",
				Scale: 0.001,
			},
			{
				// raw=2 em todos os 4 no walk. INFERIDO: 2 provavelmente
				// = "normal/online" (sem retificador reportado como
				// falhado neste momento). Confirmar mapa de códigos
				// (0/1/2/3...) — sem isso não dá para distinguir status
				// real no motor de alarmes.
				OID:   ".1.3.6.1.4.1.6302.2.1.2.11.4.1.8.3",
				Key:   "rectifier_1_status",
				Scale: 1,
			},
			{
				OID:   ".1.3.6.1.4.1.6302.2.1.2.11.4.1.8.4",
				Key:   "rectifier_2_status",
				Scale: 1,
			},
			{
				OID:   ".1.3.6.1.4.1.6302.2.1.2.11.4.1.8.5",
				Key:   "rectifier_3_status",
				Scale: 1,
			},
			{
				OID:   ".1.3.6.1.4.1.6302.2.1.2.11.4.1.8.6",
				Key:   "rectifier_4_status",
				Scale: 1,
			},
		},

		// ======================
		// ATENÇÃO CRÍTICA — tabela de alarmes nativa do equipamento
		// ======================
		// A Vertiv já expõe uma TABELA DE ALARMES ATIVOS via SNMP em
		// .1.3.6.1.4.1.6302.2.1.4.1.*, com uma linha por alarme (índice
		// 1, 2, ... — no walk havia 2 linhas, batendo exatamente com
		// "Todos os Alarmes(2)" na UI: 1 Crítico + 1 Observação):
		//
		//   .4.1.4.<idx> = severidade (6 = Crítico, 3 = Observação — INFERIDO
		//                  por cruzamento com a contagem da UI, não
		//                  confirmado contra tabela de códigos oficial)
		//   .4.1.5.<idx> = mensagem (ex.: "Rectifier Lost, its owner: Rect
		//                  Group" / "Low Capacity, its owner: Batt1")
		//   .4.1.6.<idx> = timestamp (raw ticks, formato ainda não decifrado)
		//
		// Isto é uma TABELA DINÂMICA, não um valor escalar com threshold —
		// não cabe no modelo AlarmRule abaixo (que é para thresholds sobre
		// métricas contínuas). Precisa de um walk periódico dedicado a
		// .1.3.6.1.4.1.6302.2.1.4.1 no adapter, com parsing próprio que
		// gere/feche eventos via CreateOrTouch/Resolve por alarm_key
		// (índice + mensagem), de forma semelhante ao que falta hoje no
		// SNMPIngestService para towers degraded/Fim de Vida (Prioridade
		// #1 em aberto). Ou seja: este equipamento já dá de bandeja o que
		// falta na Eltek — não desperdiçar isso implementando outro
		// caminho que também não gera evento real.
		//
		// TODO: implementar walkAlarmTable() no adapter Vertiv, mapear
		// severidade 6/3/outros valores possíveis, e ligar ao mesmo
		// pipeline de eventos/tickets usado pelos outros vendors.
		Alarms: []snmp.AlarmRule{

			// Únicos alarmes baseados em threshold que já temos confiança
			// suficiente para ativar agora (métricas escalares, não a
			// tabela dinâmica acima).

			{
				// Reaproveita o mesmo baseline empírico já validado para
				// Eltek em clima tropical — não é confirmado especificamente
				// para o Netsure, tratar como ponto de partida a ajustar
				// com dados reais de campo deste hardware.
				Key:       "system_temperature_1_c",
				Severity:  domain.EventSeverityWarning,
				Threshold: 40,
				Condition: "gt",
				Message:   "vertiv system temperature high",
			},
			{
				Key:       "system_temperature_2_c",
				Severity:  domain.EventSeverityWarning,
				Threshold: 40,
				Condition: "gt",
				Message:   "vertiv system temperature high",
			},

			// mains — mesmo cuidado do Eltek: IgnoreZero para não gerar
			// falso alarme quando o campo simplesmente não é reportado.
			{
				Key:        "mains_voltage_l1_v",
				Severity:   domain.EventSeverityWarning,
				Threshold:  80,
				Condition:  "lt",
				Message:    "vertiv mains power failure L1",
				IgnoreZero: true,
			},
			{
				Key:        "mains_voltage_l2_v",
				Severity:   domain.EventSeverityWarning,
				Threshold:  80,
				Condition:  "lt",
				Message:    "vertiv mains power failure L2",
				IgnoreZero: true,
			},
			{
				Key:        "mains_voltage_l3_v",
				Severity:   domain.EventSeverityWarning,
				Threshold:  80,
				Condition:  "lt",
				Message:    "vertiv mains power failure L3",
				IgnoreZero: true,
			},
		},
	}
}
