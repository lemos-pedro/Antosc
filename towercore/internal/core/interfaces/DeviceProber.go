package interfaces

import "context"

// ProbeResult é o resultado de uma sondagem de discovery a um único IP:
// só o essencial para decidir se há "algo" SNMP a responder ali, e que
// vendor é provável que seja, com base no sysObjectID.
type ProbeResult struct {
	Responded   bool
	SysObjectID string
}

// DeviceProber sonda um IP nu (sem Tower associada) para descobrir se
// existe um agente SNMP a responder, usando o OID padrão sysObjectID
// (1.3.6.1.2.1.1.2.0), presente em qualquer implementação SNMP
// independentemente do vendor.
//
// Esta é uma interface deliberadamente separada de Collector
// (adapters/snmp): Collector já assume uma domain.Tower e um Profile
// conhecidos; DeviceProber serve exatamente o caso em que ainda não
// sabemos nada sobre o dispositivo.
type DeviceProber interface {
	Probe(ctx context.Context, ip string, community string) (ProbeResult, error)
}
