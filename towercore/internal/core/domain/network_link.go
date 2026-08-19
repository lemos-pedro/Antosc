package domain

import "time"

// LinkOperStatus representa o estado operacional de uma interface de rede,
// espelhando os valores possíveis de ifOperStatus (IF-MIB, RFC 2863).
type LinkOperStatus string

const (
	LinkStatusUp             LinkOperStatus = "up"
	LinkStatusDown           LinkOperStatus = "down"
	LinkStatusTesting        LinkOperStatus = "testing"
	LinkStatusUnknown        LinkOperStatus = "unknown"
	LinkStatusDormant        LinkOperStatus = "dormant"
	LinkStatusNotPresent     LinkOperStatus = "not_present"
	LinkStatusLowerLayerDown LinkOperStatus = "lower_layer_down"
)

// NetworkLink representa um circuito de trânsito de fibra que termina numa
// interface de um router monitorizável via SNMP (IF-MIB).
//
// Um NetworkLink não é uma Tower: pode não estar associado 1:1 a um site,
// tem duas pontas lógicas (router local + operador remoto) e uma capacidade
// nominal contratada que serve de referência para cálculo de utilização.
type NetworkLink struct {
	LinkID            string // UUID
	Name              string // nome amigável, ex: "LUANDA-BENGUELA_10Gbps"
	RouterHost        string // hostname/identificador do router, ex: "Router Antosc ANGONAP"
	RouterIP          string // IP de gestão SNMP do router
	IfIndex           int    // ifIndex SNMP da interface neste router
	IfDescr           string // ifDescr bruto, ex: "TenGigabitEthernet2/3"
	IfAlias           string // ifAlias, ex: "CONNECTED-BEN-RT"
	Operator          string // operador/cliente do circuito, ex: "Africell", "Unitel", "" se interno
	MediaType         string // "fiber" (fixo nesta fase; rádio fica para fase 2)
	NominalCapacityMb int64  // capacidade contratada/nominal em Mbps, ex: 10000 para 10Gbps
	RegionID          string // opcional: FK para region, se o link estiver associado a uma rota regional
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// LinkMetricSnapshot é uma leitura pontual de métricas SNMP de uma interface,
// tal como devolvida pelo adapter na altura da recolha.
type LinkMetricSnapshot struct {
	LinkID         string
	CollectedAt    time.Time
	OperStatus     LinkOperStatus
	AdminStatus    LinkOperStatus
	InOctets       uint64 // contador acumulado (ifHCInOctets, 64-bit)
	OutOctets      uint64 // contador acumulado (ifHCOutOctets, 64-bit)
	InErrors       uint64
	OutErrors      uint64
	InDiscards     uint64
	OutDiscards    uint64
	SpeedMb        int64 // ifHighSpeed, em Mbps
	InUtilPercent  float64 // calculado: bps de entrada / capacidade nominal
	OutUtilPercent float64 // calculado: bps de saída / capacidade nominal
}

// LinkEvent representa uma falha ou degradação detetada num link,
// equivalente ao domain.Event já usado para torres, mas com contexto
// específico de rede (erros de interface, discards, down).
type LinkEvent struct {
	EventID    string
	LinkID     string
	Type       string // "down", "flapping", "high_errors", "high_utilization", "degraded"
	Severity   string // "info" | "warning" | "critical"
	Message    string
	OccurredAt time.Time
	ResolvedAt *time.Time
	AlarmKey   string // para deduplicação, mesmo padrão CreateOrTouch/Resolve já usado
}
