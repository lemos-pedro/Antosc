// Package domain contém as entidades centrais do domínio de negócio.
package domain

import (
	"context"
	"github.com/google/uuid"
	"time"
)

// RadioKPI representa indicadores-chave de desempenho (KPIs) de rádio
// federados de sistemas NMS/EMS externos já processados.
// Esta tabela permite correlacionar métricas de infraestrutura do TowerCore
// com qualidade de serviço de rádio real entregue aos usuários.
type RadioKPI struct {
	ID            uuid.UUID `json:"radio_kpi_id"`
	TowerID       uuid.UUID `json:"tower_id"`
	SectorID      string    `json:"sector_id,omitempty"` // ex: "A", "B", "C" (vazio para setor omnidirecional)
	CellTechnique string    `json:"cell_technique"`      // LTE, 5GNR, GSM, UMTS, NR

	// Timestamp da medição no equipamento de rádio
	MeasuredAt time.Time `json:"measured_at"`
	ReceivedAt time.Time `json:"received_at"` // Quando chegou ao nosso sistema

	// === MÉTRICAS DE POTÊNCIA E QUALIDADE ===
	TxPowerWatt *float64 `json:"tx_power_watt,omitempty"` // Potência de transmissão em Watts
	TxPowerDbm  *float64 `json:"tx_power_dbm,omitempty"`  // Potência de transmissão em dBm
	RxPowerDbm  *float64 `json:"rx_power_dbm,omitempty"`  // Potência média de sinal recebido em dBm
	SnrDb       *float64 `json:"snr_db,omitempty"`        // Signal-to-Noise Ratio em dB
	SinrDb      *float64 `json:"sinr_db,omitempty"`       // Signal-to-Interference-plus-Noise Ratio em dB
	RsrpDbm     *float64 `json:"rsrp_dbm,omitempty"`      // LTE/5G: Reference Signal Received Power
	RsrqDbm     *float64 `json:"rsrq_db,omitempty"`       // LTE/5G: Reference Signal Received Quality

	// === MÉTRICAS DE UTILIZAÇÃO E CAPACIDADE ===
	ConnectedUEs        *int     `json:"connected_ues,omitempty"`         // Número de User Equipments conectados
	MaxSupportedUEs     *int     `json:"max_supported_ues,omitempty"`     // Capacidade máxima teórica do setor
	PRBUtilizationPct   *float64 `json:"prb_utilization_pct,omitempty"`   // LTE/5G: Physical Resource Block utilization (%)
	ChannelOccupancyPct *float64 `json:"channel_occupancy_pct,omitempty"` // GSM/UMTS: Channel occupancy (%)

	// === MÉTRICAS DE ERROS E QUALIDADE ===
	Ber          *float64 `json:"ber,omitempty"`            // Bit Error Rate
	Bler         *float64 `json:"bler,omitempty"`           // Block Error Rate
	Fer          *float64 `json:"fer,omitempty"`            // Frame Error Rate
	CodecDropPct *float64 `json:"codec_drop_pct,omitempty"` // Percentual de pacotes descartados por falha de codec

	// === MÉTRICAS DE MOBILIDADE ===
	HoAttempt  *int `json:"ho_attempt,omitempty"`   // Número de tentativas de handover
	HoSuccess  *int `json:"ho_success,omitempty"`   // Número de handovers bem-sucedidos
	HoFail     *int `json:"ho_fail,omitempty"`      // Número de handovers que falharam
	HoPingPong *int `json:"ho_ping_pong,omitempty"` // Handovers ping-pong (setor A->B->A em curto tempo)

	// === MÉTRICAS DE DISPONIBILIDADE E EXPERIÊNCIA ===
	CallDropPct    *float64 `json:"call_drop_pct,omitempty"`     // Percentual de chamadas que caíram
	CallBlockPct   *float64 `json:"call_block_pct,omitempty"`    // Percentual de chamadas bloqueadas por falta de recursos
	PDCPSDULossPct *float64 `json:"pdcp_sdu_loss_pct,omitempty"` // Perdida de pacotes na camada PDCP
	RLCRetransPct  *float64 `json:"rlc_retrans_pct,omitempty"`   // Retransmissões na camada RLC

	// === METADADOS ===
	SourceSystem          string `json:"source_system"`           // Qual sistema NMS/EMS enviou estes dados
	CollectionIntervalSec int    `json:"collection_interval_sec"` // Intervalo de coleta destes KPIs (ex: 300 = 5 min)
	RawData               []byte `json:"raw_data,omitempty"`      // Dados brutos originais para auditoria/debug (opcional)

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RadioKPIFilter define critérios para filtrar e consultar KPIs de rádio.
type RadioKPIFilter struct {
	TowerID              uuid.UUID
	SectorID             *string // Pointer para diferenciar entre vazio explícito e não definido
	CellTechnique        *string
	MeasuredAtAfter      *time.Time
	MeasuredAtBefore     *time.Time
	ReceivedAtAfter      *time.Time
	ReceivedAtBefore     *time.Time
	MinRxPowerDbm        *float64
	MaxRxPowerDbm        *float64
	MinSnrDb             *float64
	MaxSnrDb             *float64
	MinPrbUtilizationPct *float64
	MaxPrbUtilizationPct *float64
	Limit                int
	Offset               int
	OrderBy              []string // ex: []string{"measured_at DESC", "tower_id ASC"}
}

// RadioKPIGenericRepository define a interface genérica para repositórios de KPIs de rádio.
// Esta abstração permite trocar facilmente a implementação (PostgreSQL, in-memory para teste, etc.).
type RadioKPIGenericRepository interface {
	// CreateMany insere múltiplos KPIs de rádio de uma vez (ótimo para batch de ETLs)
	CreateMany(ctx context.Context, kpis []*RadioKPI) error
	// List retorna KPIs de rádio baseado em filtros
	List(ctx context.Context, filter *RadioKPIFilter) ([]*RadioKPI, int, error)
	// DeleteOlderThan remove KPIs mais antigos que um determinado tempo (para retenção)
	DeleteOlderThan(ctx context.Context, olderThan time.Time) error
	// GetByTowerAndSector retorna o KPI mais recente para uma torre/setor/técnica específica
	GetByTowerAndSector(ctx context.Context, towerID uuid.UUID, sectorID string, technique string) (*RadioKPI, error)
}
