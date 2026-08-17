// Package domain contém as entidades centrais do domínio de negócio.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// BackhaulInterface representa métricas coletadas de uma interface de backhaul
// de um site (torre). Estas métricas podem vir de polling SNMP, scripts locais,
// ou agentes instalados no equipamento de agregação.
type BackhaulInterface struct {
	ID          uuid.UUID `json:"backhaul_interface_id"`
	TowerID     uuid.UUID `json:"tower_id"`
	InterfaceID string    `json:"interface_id"` // UUID gerado para identificar esta interface específica nesta torre

	// Identificação da interface
	Name        string    `json:"interface_name"`        // ex: "eth0", "mgmt0", "bundle-ether1"
	Description string    `json:"interface_description"` // Descrição textual da interface (opcional)

	// Timestamp da medição
	MeasuredAt  time.Time `json:"measured_at"`
	ReceivedAt  time.Time `json:"received_at"` // Quando recebido pelo nosso sistema

	// === STATUS DA INTERFACE ===
	AdminStatus string    `json:"admin_status"`   // up/down/testing/unknown (do IF-MIB ifAdminStatus)
	OperStatus  string    `json:"oper_status"`    // up/down/testing/unknown/dormant/notPresent/downLowerLayer (do IF-MIB ifOperStatus)
	LastChange  *time.Time `json:"last_change,omitempty"` // Quando a interface mudou estado pela última vez (ifLastChange)

	// === TIPO E CAPACIDADE ===
	IfType      string    `json:"if_type,omitempty"`     // Tipo da interface (ianaIfType: ethernetCsmacd(6), ieee8023adLag(137), etc.)
	IfSpeedMbps *float64  `json:"if_speed_mbps,omitempty"` // Speed in Mbps
	DuplexMode  *string   `json:"duplex_mode,omitempty"` // full/half/unknown
	MediaType   *string   `json:"media_type,omitempty"`  // Tipo de meio: fiber, copper, wireless, etc.
	ConnectorType *string  `json:"connector_type,omitempty"` // Tipo de conector: LC, SC, RJ45, etc.

	// === CONTADORES DE PACOTES E BYTES (IF-MIB) ===
	InOctets    *uint64   `json:"in_octets,omitempty"`   // Bytes recebidos (ifInOctets)
	OutOctets   *uint64   `json:"out_octets,omitempty"`  // Bytes enviados (ifOutOctets)
	InUnicastPkts *uint64  `json:"in_unicast_pkts,omitempty"` // Pacotes unicast recebidos (ifInUcastPkts)
	OutUnicastPkts *uint64  `json:"out_unicast_pkts,omitempty"` // Pacotes unicast enviados (ifOutUcastPkts)
	InDiscards  *uint64   `json:"in_discards,omitempty"` // Pacotes descartados na entrada (ifInDiscards)
	OutDiscards *uint64   `json:"out_discards,omitempty"` // Pacotes descartados na saída (ifOutDiscards)
	InErrors    *uint64   `json:"in_errors,omitempty"`   // Erros na entrada (ifInErrors)
	OutErrors   *uint64   `json:"out_errors,omitempty"`  // Erros na saída (ifOutErrors)
	InUnknownProtos *uint64 `json:"in_unknown_protos,omitempty"` // Pacotes descartados por protocolo desconhecido (ifInUnknownProtos)

	// === CONTADORES DETALHADOS (SE DISPONÍVEL) ===
	InFrameErrors *uint64   `json:"in_frame_errors,omitempty"` // Erros de frame (alignment, CRC, etc.)
	OutFrameErrors *uint64   `json:"out_frame_errors,omitempty"`
	InJabbers     *uint64   `json:"in_jabbers,omitempty"`    // Pacotes muito grandes com CRC ruim
	OutJabbers    *uint64   `json:"out_jabbers,omitempty"`
	InFragments   *uint64   `json:"in_fragments,omitempty"`  // Pacotes muito pequenos
	OutFragments  *uint64   `json:"out_fragments,omitempty"`

	// === UTILIZAÇÃO E PERFORMANCE (CALCULADO OU MEDIDO ATIVAMENTE) ===
	UtilizationPct    *float64  `json:"utilization_pct,omitempty"`    // Utilização da banda em %
	InBandwidthMbps   *float64  `json:"in_bandwidth_mbps,omitempty"`  // Banda larga de entrada em Mbps
	OutBandwidthMbps  *float64   `json:"out_bandwidth_mbps,omitempty"` // Banda larga de saída em Mbps

	// === MEDIÇÕES ATIVAS DE LATÊNCIA / LOSS / JITTER (OPCIONAL) ===
	AvgLatencyMs      *float64  `json:"avg_latency_ms,omitempty"`     // Latência média de ida e volta em milissegundos
	MinLatencyMs      *float64  `json:"min_latency_ms,omitempty"`
	MaxLatencyMs      *float64  `json:"max_latency_ms,omitempty"`
	LossPct           *float64  `json:"loss_pct,omitempty"`           // Percentual de perda de pacotes
	JitterMs          *float64  `json:"jitter_ms,omitempty"`          // Variação na latência (jitter)

	// === MÉTRICAS ÓTICAS ESFSENCIAIS PARA O TÉCNICO DE CAMPO ===
	// Estas métricas permitem ao técnico de campo diagnosticar problemas físicos de fibra ótica
	TxPowerDbm        *float64  `json:"tx_power_dbm,omitempty"`       // Potência de transmissão óptica (dBm)
	RxPowerDbm        *float64  `json:"rx_power_dbm,omitempty"`       // Potência de recebimento óptico (dBm)
	OpticTempC        *float64  `json:"optic_temp_c,omitempty"`       // Temperatura do transceptor óptico (°C)
	OpticBiasCurrentMa*float64  `json:"optic_bias_current_ma,omitempty"` // Corrente de bias do laser (mA) - indicador de envelhecimento
	LosEvents         *uint64   `json:"los_events,omitempty"`         // Contagem de eventos Loss of Signal
	LofEvents         *uint64   `json:"lof_events,omitempty"`         // Contagem de eventos Loss of Frame
	LomEvents         *uint64   `json:"lom_events,omitempty"`         // Contagem de eventos Loss of Multiframe
	OpticWavelengthNm *int      `json:"optic_wavelength_nm,omitempty"` // Comprimento de onda nominal (nm) - ex: 850, 1310, 1550
	OpticVendor       *string   `json:"optic_vendor,omitempty"`       // Vendor do transceptor (ex: Finisar, II-VI)
	OpticPartNumber   *string   `json:"optic_part_number,omitempty"`  // Número de peça (para reposição exata)
	OpticSerialNumber *string   `json:"optic_serial_number,omitempty"` // Número de série (garantia e histórico)
	OpticDateCode     *string   `json:"optic_date_code,omitempty"`    // Data de fabricação (YYWW ou YYYYMM)

	// === METADADOS ===
	SourcePoller      string    `json:"source_poller"`                // Quem coletou estes dados (ex: "local_agent", "snmp_poller", "script")
	CollectionIntervalSec int     `json:"collection_interval_sec"`      // Intervalo em segundos desta medição
	RawData           []byte    `json:"raw_data,omitempty"`           // Dados brutos originais (para debug/audit)

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BackhaulInterfaceFilter define critérios para filtrar e consultar interfaces de backhaul.
type BackhaulInterfaceFilter struct {
	TowerID         uuid.UUID
	InterfaceName   *string
	MeasuredAtAfter *time.Time
	MeasuredAtBefore *time.Time
	ReceivedAtAfter *time.Time
	ReceivedAtBefore *time.Time
	AdminStatus     *string
	OperStatus      *string
	MinUtilizationPct *float64
	MaxUtilizationPct *float64
	MinLatencyMs   *float64
	MaxLatencyMs   *float64
	MaxLossPct     *float64
	Limit          int
	Offset         int
	OrderBy        []string // ex: []string{"measured_at DESC", "tower_id ASC"}
}

// BackhaulInterfaceRepository define a interface para repositórios de métricas de backhaul.
type BackhaulInterfaceRepository interface {
	// Create insere uma nova medição de interface de backhaul
	Create(ctx context.Context, iface *BackhaulInterface) error
	// CreateMany insere múltiplas medições de uma vez (ótimo para batch)
	CreateMany(ctx context.Context, ifaces []*BackhaulInterface) error
	// List retorna medições baseado em filtros
	List(ctx context.Context, filter *BackhaulInterfaceFilter) ([]*BackhaulInterface, int, error)
	// GetLatest retorna a medição mais recente para uma torre/interface
	GetLatest(ctx context.Context, towerID uuid.UUID, interfaceName string) (*BackhaulInterface, error)
	// DeleteOlderThan remove medições mais antigas que um determinado tempo (para retenção)
	DeleteOlderThan(ctx context.Context, olderThan time.Time) error
}