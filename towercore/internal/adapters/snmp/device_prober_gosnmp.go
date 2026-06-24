package snmp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"

	"towercore/internal/core/interfaces"
)

// sysObjectIDOID é o OID padrão SNMPv2-MIB::sysObjectID, presente em
// qualquer agente SNMP (v1 ou v2c), independentemente do vendor.
// O valor devolvido normalmente começa por 1.3.6.1.4.1.<enterprise>,
// onde <enterprise> identifica o fabricante (ex: 2011=Huawei,
// 12148=Eltek, 53318=Enetek).
const sysObjectIDOID = "1.3.6.1.2.1.1.2.0"

// GoSNMPProber implementa interfaces.DeviceProber usando gosnmp,
// sempre em SNMPv2c (o discovery não tenta v1 nem v3 — equipamento
// que só fale essas versões não é encontrado por este prober).
type GoSNMPProber struct {
	timeout time.Duration
	retries int
	port    uint16
}

// NewGoSNMPProber cria um prober com defaults curtos: discovery varre
// muitos IPs, a maioria dos quais não vai responder, por isso o
// timeout deve ser bem mais curto que o do polling normal para não
// tornar um scan de /24 demasiado lento.
func NewGoSNMPProber(timeout time.Duration, retries int) *GoSNMPProber {
	if timeout <= 0 {
		timeout = 800 * time.Millisecond
	}
	if retries < 0 {
		retries = 0
	}
	return &GoSNMPProber{timeout: timeout, retries: retries, port: 161}
}

// Probe implementa interfaces.DeviceProber.
func (p *GoSNMPProber) Probe(ctx context.Context, ip string, community string) (interfaces.ProbeResult, error) {
	if strings.TrimSpace(ip) == "" {
		return interfaces.ProbeResult{}, errors.New("ip is required")
	}
	if strings.TrimSpace(community) == "" {
		return interfaces.ProbeResult{}, errors.New("community is required")
	}

	client := &gosnmp.GoSNMP{
		Target:    ip,
		Port:      p.port,
		Community: community,
		Version:   gosnmp.Version2c,
		Timeout:   p.timeout,
		Retries:   p.retries,
		MaxOids:   gosnmp.MaxOids,
	}

	if err := client.Connect(); err != nil {
		return interfaces.ProbeResult{}, fmt.Errorf("connect: %w", err)
	}
	defer client.Conn.Close()

	type result struct {
		packet *gosnmp.SnmpPacket
		err    error
	}
	ch := make(chan result, 1)

	go func() {
		packet, err := client.Get([]string{sysObjectIDOID})
		ch <- result{packet: packet, err: err}
	}()

	select {
	case <-ctx.Done():
		return interfaces.ProbeResult{}, ctx.Err()
	case r := <-ch:
		if r.err != nil {
			// Timeout ou recusa são o resultado normal e esperado para
			// a maioria dos IPs durante um scan — não é um erro do
			// ponto de vista do discovery, só significa "não respondeu".
			return interfaces.ProbeResult{Responded: false}, nil
		}
		if len(r.packet.Variables) == 0 {
			return interfaces.ProbeResult{Responded: false}, nil
		}

		sysObjectID := formatOID(r.packet.Variables[0])
		return interfaces.ProbeResult{
			Responded:   true,
			SysObjectID: sysObjectID,
		}, nil
	}
}

// formatOID extrai o valor de sysObjectID como string, independentemente
// de o gosnmp o devolver como ObjectIdentifier (string) ou como bytes.
func formatOID(pdu gosnmp.SnmpPDU) string {
	switch v := pdu.Value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
