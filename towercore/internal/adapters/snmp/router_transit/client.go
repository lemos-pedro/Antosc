package router_transit

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
	"towercore/internal/infrastructure/logger"
)

// Client implementa interfaces.LinkMonitor usando SNMP v2c contra o IF-MIB
// padrão de um router. Não depende de nenhum OID vendor-specific.
type Client struct {
	community      string
	port           uint16
	timeoutSeconds int
	retries        int
	log            *logger.Logger
}

// NewClient cria um cliente para routers de trânsito. community e port
// seguem o mesmo padrão de configuração usado nos outros adapters SNMP
// (SNMP_TIMEOUT_SECONDS / SNMP_RETRIES já existentes em configuration.md).
func NewClient(community string, port uint16, timeoutSeconds, retries int, log *logger.Logger) *Client {
	if port == 0 {
		port = 161
	}
	if timeoutSeconds == 0 {
		timeoutSeconds = 5
	}
	return &Client{
		community:      community,
		port:           port,
		timeoutSeconds: timeoutSeconds,
		retries:        retries,
		log:            log,
	}
}

func (c *Client) newSession(routerIP string) *gosnmp.GoSNMP {
	return &gosnmp.GoSNMP{
		Target:    routerIP,
		Port:      c.port,
		Community: c.community,
		Version:   gosnmp.Version2c,
		Timeout:   time.Duration(c.timeoutSeconds) * time.Second,
		Retries:   c.retries,
	}
}

// GetInterfaceMetrics faz Get pontual dos OIDs IF-MIB relevantes para o
// ifIndex indicado. Devolve contadores brutos — o cálculo de bps/utilização
// fica a cargo do core/services, que compara com o snapshot anterior.
func (c *Client) GetInterfaceMetrics(ctx context.Context, routerIP string, ifIndex int) (*domain.LinkMetricSnapshot, error) {
	session := c.newSession(routerIP)
	if err := session.Connect(); err != nil {
		return nil, fmt.Errorf("router_transit: falha ao conectar a %s: %w", routerIP, err)
	}
	defer session.Conn.Close()

	suffix := fmt.Sprintf(".%d", ifIndex)
	oids := []string{
		OIDIfOperStatus + suffix,
		OIDIfAdminStatus + suffix,
		OIDIfInErrors + suffix,
		OIDIfOutErrors + suffix,
		OIDIfInDiscards + suffix,
		OIDIfOutDiscards + suffix,
		OIDIfHCInOctets + suffix,
		OIDIfHCOutOctets + suffix,
		OIDIfHighSpeed + suffix,
	}

	result, err := session.Get(oids)
	if err != nil {
		return nil, fmt.Errorf("router_transit: falha no SNMP Get para %s ifIndex %d: %w", routerIP, ifIndex, err)
	}

	snapshot := &domain.LinkMetricSnapshot{
		CollectedAt: time.Now().UTC(),
	}

	for _, variable := range result.Variables {
		name := strings.TrimPrefix(variable.Name, ".")
		switch {
		case strings.HasPrefix(name, strings.TrimPrefix(OIDIfOperStatus, ".")):
			snapshot.OperStatus = domain.LinkOperStatus(operStatusIntToString[gosnmp.ToBigInt(variable.Value).IntPart()])
		case strings.HasPrefix(name, strings.TrimPrefix(OIDIfAdminStatus, ".")):
			snapshot.AdminStatus = domain.LinkOperStatus(operStatusIntToString[gosnmp.ToBigInt(variable.Value).IntPart()])
		case strings.HasPrefix(name, strings.TrimPrefix(OIDIfInErrors, ".")):
			snapshot.InErrors = gosnmp.ToBigInt(variable.Value).Uint64()
		case strings.HasPrefix(name, strings.TrimPrefix(OIDIfOutErrors, ".")):
			snapshot.OutErrors = gosnmp.ToBigInt(variable.Value).Uint64()
		case strings.HasPrefix(name, strings.TrimPrefix(OIDIfInDiscards, ".")):
			snapshot.InDiscards = gosnmp.ToBigInt(variable.Value).Uint64()
		case strings.HasPrefix(name, strings.TrimPrefix(OIDIfOutDiscards, ".")):
			snapshot.OutDiscards = gosnmp.ToBigInt(variable.Value).Uint64()
		case strings.HasPrefix(name, strings.TrimPrefix(OIDIfHCInOctets, ".")):
			snapshot.InOctets = gosnmp.ToBigInt(variable.Value).Uint64()
		case strings.HasPrefix(name, strings.TrimPrefix(OIDIfHCOutOctets, ".")):
			snapshot.OutOctets = gosnmp.ToBigInt(variable.Value).Uint64()
		case strings.HasPrefix(name, strings.TrimPrefix(OIDIfHighSpeed, ".")):
			snapshot.SpeedMb = gosnmp.ToBigInt(variable.Value).Int64()
		}
	}

	if snapshot.OperStatus == "" {
		snapshot.OperStatus = domain.LinkStatusUnknown
		c.log.Errorf("router_transit: ifOperStatus vazio para %s ifIndex %d — possível ifIndex incorreto", routerIP, ifIndex)
	}

	return snapshot, nil
}

// DiscoverInterfaces faz um SNMP Walk a ifDescr/ifAlias/ifType para permitir
// mapear novas interfaces sem introduzir ifIndex manualmente. Equivalente
// funcional ao LLD do Zabbix, mas disparado sob demanda (não automático).
func (c *Client) DiscoverInterfaces(ctx context.Context, routerIP string) ([]interfaces.InterfaceDescriptor, error) {
	session := c.newSession(routerIP)
	if err := session.Connect(); err != nil {
		return nil, fmt.Errorf("router_transit: falha ao conectar a %s: %w", routerIP, err)
	}
	defer session.Conn.Close()

	descrByIndex := map[int]string{}
	aliasByIndex := map[int]string{}
	typeByIndex := map[int]int{}

	err := session.BulkWalk(OIDIfDescr, func(pdu gosnmp.SnmpPDU) error {
		idx, ifIndex := parseIfIndexFromOID(pdu.Name, OIDIfDescr)
		if idx {
			descrByIndex[ifIndex] = pduToString(pdu)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("router_transit: falha no walk ifDescr em %s: %w", routerIP, err)
	}

	err = session.BulkWalk(OIDIfAlias, func(pdu gosnmp.SnmpPDU) error {
		idx, ifIndex := parseIfIndexFromOID(pdu.Name, OIDIfAlias)
		if idx {
			aliasByIndex[ifIndex] = pduToString(pdu)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("router_transit: falha no walk ifAlias em %s: %w", routerIP, err)
	}

	err = session.BulkWalk(OIDIfType, func(pdu gosnmp.SnmpPDU) error {
		idx, ifIndex := parseIfIndexFromOID(pdu.Name, OIDIfType)
		if idx {
			typeByIndex[ifIndex] = int(gosnmp.ToBigInt(pdu.Value).Int64())
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("router_transit: falha no walk ifType em %s: %w", routerIP, err)
	}

	descriptors := make([]interfaces.InterfaceDescriptor, 0, len(descrByIndex))
	for idx, descr := range descrByIndex {
		descriptors = append(descriptors, interfaces.InterfaceDescriptor{
			IfIndex: idx,
			IfDescr: descr,
			IfAlias: aliasByIndex[idx],
			IfType:  typeByIndex[idx],
		})
	}

	return descriptors, nil
}

func parseIfIndexFromOID(oidName, baseOID string) (bool, int) {
	trimmedBase := strings.TrimPrefix(baseOID, ".")
	trimmedName := strings.TrimPrefix(oidName, ".")
	if !strings.HasPrefix(trimmedName, trimmedBase) {
		return false, 0
	}
	suffix := strings.TrimPrefix(trimmedName, trimmedBase+".")
	var ifIndex int
	if _, err := fmt.Sscanf(suffix, "%d", &ifIndex); err != nil {
		return false, 0
	}
	return true, ifIndex
}

func pduToString(pdu gosnmp.SnmpPDU) string {
	switch pdu.Value.(type) {
	case []byte:
		return string(pdu.Value.([]byte))
	default:
		return fmt.Sprintf("%v", pdu.Value)
	}
}
