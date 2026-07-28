package neteco

import (
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/gosnmp/gosnmp"
)

const (
	// OID padrão usado pelo SNMPv2 para identificar o tipo do Trap.
	snmpTrapOID = "1.3.6.1.6.3.1.1.4.1.0"

	// Huawei Enterprise OID.
	huaweiEnterpriseOID = "1.3.6.1.4.1.2011"

	// OIDs do corpo do trap de alarme Huawei (NetEco NBI).
	// Confirmados empiricamente via captura de traps reais em 2026-07-24.
	oidAlarmEventType  = "1.3.6.1.4.1.2011.2.15.2.4.3.3.2.0"  // 1=ativo, 2=resolvido
	oidAlarmOccurTime  = "1.3.6.1.4.1.2011.2.15.2.4.3.3.3.0"
	oidAlarmSiteName   = "1.3.6.1.4.1.2011.2.15.2.4.3.3.4.0"
	oidAlarmNEID       = "1.3.6.1.4.1.2011.2.15.2.4.3.3.7.0"
	oidAlarmNo         = "1.3.6.1.4.1.2011.2.15.2.4.3.3.8.0"
	oidAlarmSeverity   = "1.3.6.1.4.1.2011.2.15.2.4.3.3.11.0" // TODO: validar escala real contra painel NetEco
	oidAlarmClearTime  = "1.3.6.1.4.1.2011.2.15.2.4.3.3.15.0"
	oidAlarmSiteIDInfo = "1.3.6.1.4.1.2011.2.15.2.4.3.3.27.0"
	oidAlarmDesc       = "1.3.6.1.4.1.2011.2.15.2.4.3.3.28.0"
)

// TrapEvent representa um alarme Huawei recebido via SNMP Trap.
type TrapEvent struct {
	EventType   int
	SiteName    string
	Severity    int
	NEID        string
	AlarmNo     string
	Description string

	// OID original do Trap.
	TrapOID string

	// IP do equipamento que enviou o Trap.
	SourceIP string
}

// TrapHandler processa um TrapEvent já parseado.
type TrapHandler func(TrapEvent)

// StartTrapListener inicia o listener SNMP Trap.
func StartTrapListener(
	port uint16,
	community string,
	handler TrapHandler,
) error {
	if handler == nil {
		return fmt.Errorf("neteco trap handler não pode ser nil")
	}

	tl := gosnmp.NewTrapListener()

	tl.OnNewTrap = func(
		packet *gosnmp.SnmpPacket,
		addr *net.UDPAddr,
	) {
		sourceIP := ""

		if addr != nil {
			sourceIP = addr.IP.String()
		}

		log.Printf(
			"[NetEco][trap] recebido source=%s variables=%d community=%s",
			sourceIP,
			len(packet.Variables),
			packet.Community,
		)

		event, ok := ParseTrap(packet, sourceIP)

		if !ok {
			log.Printf(
				"[NetEco][trap] trap ignorado source=%s",
				sourceIP,
			)
			return
		}

		log.Printf(
			"[NetEco][trap] alarme reconhecido site=%s neid=%s alarm=%s severity=%d eventType=%d desc=%q",
			event.SiteName,
			event.NEID,
			event.AlarmNo,
			event.Severity,
			event.EventType,
			event.Description,
		)

		handler(event)
	}

	params := &gosnmp.GoSNMP{
		Port:      port,
		Community: community,
		Version:   gosnmp.Version2c,
		Logger: gosnmp.NewLogger(
			log.New(
				log.Writer(),
				"[neteco-trap] ",
				0,
			),
		),
	}

	tl.Params = params

	return tl.Listen(
		fmt.Sprintf(
			"0.0.0.0:%d",
			port,
		),
	)
}

// ParseTrap analisa um SNMP Trap Huawei.
// Exportamos a função para permitir testes unitários.
func ParseTrap(
	packet *gosnmp.SnmpPacket,
	sourceIP string,
) (TrapEvent, bool) {
	if packet == nil {
		log.Printf(
			"[NetEco][trap] packet nil",
		)

		return TrapEvent{}, false
	}

	var event TrapEvent

	event.SourceIP = sourceIP

	// Guardamos todos os valores temporariamente.
	// Isso permite identificar melhor o formato real do Trap.
	var values []trapVariable

	for _, variable := range packet.Variables {
		oid := normalizeOID(variable.Name)

		value := toString(variable.Value)

		log.Printf(
			"[NetEco][trap] variable oid=%s type=%v value=%q",
			oid,
			variable.Type,
			value,
		)

		values = append(
			values,
			trapVariable{
				OID:   oid,
				Type:  variable.Type,
				Value: variable.Value,
			},
		)

		// OID padrão SNMPv2:
		//
		// 1.3.6.1.6.3.1.1.4.1.0
		//
		// contém o OID específico do Trap.
		if oid == snmpTrapOID {
			event.TrapOID = normalizeOID(value)
		}
	}

	// Heartbeat SNMP Agent.
	//
	// Padrão observado: SiteName="SNMP Agent", valor fixo "60",
	// enviado a cada ~60s. Não é um alarme Huawei.
	if isHeartbeat(values) {
		log.Printf(
			"[NetEco][trap] heartbeat SNMP Agent ignorado",
		)

		return TrapEvent{}, false
	}

	// Neste momento, tentamos extrair os campos Huawei conhecidos.
	//
	// IMPORTANTE:
	// não dependemos apenas do penúltimo número do OID.
	// Primeiro tentamos reconhecer o OID completo.
	extractHuaweiFields(
		&event,
		values,
	)

	// Se ainda não temos o mínimo necessário,
	// não enviamos um evento incompleto para a camada de negócio.
	if event.NEID == "" || event.AlarmNo == "" {
		log.Printf(
			"[NetEco][trap] trap Huawei sem dados suficientes: trapOID=%s neid=%q alarmNo=%q site=%q severity=%d eventType=%d",
			event.TrapOID,
			event.NEID,
			event.AlarmNo,
			event.SiteName,
			event.Severity,
			event.EventType,
		)

		return TrapEvent{}, false
	}

	return event, true
}

type trapVariable struct {
	OID   string
	Type  gosnmp.Asn1BER
	Value interface{}
}

// extractHuaweiFields extrai os campos do Trap Huawei de alarme (NetEco NBI).
//
// OIDs confirmados empiricamente contra traps reais capturados em 2026-07-24
// (alarme "Compressor Fault" no site UIUIG002_KIMPAVITA, NE=33556777).
func extractHuaweiFields(
	event *TrapEvent,
	variables []trapVariable,
) {
	for _, variable := range variables {
		switch variable.OID {
		case oidAlarmEventType:
			// 1 = alarme ativo (raised), 2 = alarme resolvido (cleared)
			event.EventType = toInt(variable.Value)

		case oidAlarmSiteName:
			event.SiteName = toString(variable.Value)

		case oidAlarmNEID:
			event.NEID = toString(variable.Value)

		case oidAlarmNo:
			event.AlarmNo = toString(variable.Value)

		case oidAlarmSeverity:
			// TODO: validar esta escala contra o painel de alarmes do
			// NetEco antes de usar para classificar severidade em produção.
			// Valores observados até agora: 6 (ativo), 2 (limpo).
			event.Severity = toInt(variable.Value)

		case oidAlarmDesc:
			event.Description = toString(variable.Value)

		// Campos reconhecidos mas ainda não mapeados para o domínio:
		// oidAlarmOccurTime, oidAlarmClearTime, oidAlarmSiteIDInfo
		}
	}
}

// isHeartbeat identifica o heartbeat SNMP Agent.
func isHeartbeat(
	variables []trapVariable,
) bool {
	for _, variable := range variables {
		value := strings.TrimSpace(
			strings.ToLower(
				toString(variable.Value),
			),
		)

		if value == "snmp agent" {
			return true
		}
	}

	return false
}

// normalizeOID remove pontos duplicados e o ponto inicial.
func normalizeOID(oid string) string {
	return strings.TrimPrefix(
		strings.TrimSpace(oid),
		".",
	)
}

func toString(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case []byte:
		return string(v)

	case string:
		return v

	default:
		return fmt.Sprintf("%v", v)
	}
}

func toInt(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case int8:
		return int(n)
	case int16:
		return int(n)
	case int32:
		return int(n)
	case int64:
		return int(n)

	case uint:
		return int(n)
	case uint8:
		return int(n)
	case uint16:
		return int(n)
	case uint32:
		return int(n)
	case uint64:
		return int(n)

	case string:
		var value int
		if _, err := fmt.Sscanf(strings.TrimSpace(n), "%d", &value); err == nil {
			return value
		}

	case []byte:
		var value int
		if _, err := fmt.Sscanf(strings.TrimSpace(string(n)), "%d", &value); err == nil {
			return value
		}
	}

	return 0
}