// Package domain contém as entidades centrais do domínio de negócio.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// SiteEnvironment representa medições de ambiente e condições físicas do site
// (tower, shelter, cabinet). Estas métricas ajudam o técnico de campo a
// identificar riscos immediatos como superaquecimento, entrada de água,
// falha de energia, porta aberta, fumaça, etc.
type SiteEnvironment struct {
	ID     uuid.UUID `json:"site_environment_id"`
	SiteID uuid.UUID `json:"site_id"` // UUID da torre/cabinet (referência à tabela towers)

	// Timestamp da medição
	MeasuredAt time.Time `json:"measured_at"`
	ReceivedAt time.Time `json:"received_at"` // Quando recebido pelo nosso sistema

	// === TEMPERATURA INTERNA (CRÍTICA PARA EQUIPAMENTOS) ===
	InternalTempC  *float64 `json:"internal_temp_c,omitempty"`   // °C - ar dentro do shelter
	TempRiskPointC *float64 `json:"temp_risk_point_c,omitempty"` // °C - ponto crítico (ex: perto de gerador, baterias)
	TempDeltaC     *float64 `json:"temp_delta_c,omitempty"`      // °C - diferença entre ponto risco e ambiente

	// === UMIDADE RELATIVA (RISCO DE CORROSÃO E CONDENSAÇÃO) ===
	HumidityPct          *float64 `json:"humidity_pct,omitempty"`            // % RH
	HumidityRiskPointPct *float64 `json:"humidity_risk_point_pct,omitempty"` // % RH - ponto crítico

	// === PORTA DO SHELTER/CABINET ===
	DoorOpen          *bool `json:"door_open,omitempty"`           // true = porta aberta
	DoorOpenSecondary *bool `json:"door_open_secondary,omitempty"` // porta secundária (opcional)

	// === ENERGIA ===
	MainsPowerOk     *bool    `json:"mains_power_ok,omitempty"`     // true = energia da rede presente e estável
	UpsOnBattery     *bool    `json:"ups_on_battery,omitempty"`     // true = UPS está em modo bateria
	UpsBatteryPct    *float64 `json:"ups_battery_pct,omitempty"`    // % de carga da bateria do UPS
	UpsLoadPct       *float64 `json:"ups_load_pct,omitempty"`       // % de carga atual do UPS
	GeneratorRunning *bool    `json:"generator_running,omitempty"`  // true = gerador em funcionamento
	GeneratorLoadPct *float64 `json:"generator_load_pct,omitempty"` // % de carga no gerador
	GeneratorFuelPct *float64 `json:"generator_fuel_pct,omitempty"` // % de combustível restante
	MainsVoltageV    *float64 `json:"mains_voltage_v,omitempty"`    // Volts AC da rede
	MainsFrequencyHz *float64 `json:"mains_frequency_hz,omitempty"` // Hertz da rede

	// === DETECÇÃO DE FUMAÇA / INCÊNDIO ===
	SmokeDetected      *bool `json:"smoke_detected,omitempty"`        // true = fumaça detectada
	SmokeDetectedZone2 *bool `json:"smoke_detected_zone_2,omitempty"` // zona 2 (opcional)
	SmokeDetectedZone3 *bool `json:"smoke_detected_zone_3,omitempty"` // zona 3 (opcional)

	// === VIBRAÇÃO / IMPACTO ===
	VibrationDetected *bool `json:"vibration_detected,omitempty"` // true = vibração anormal detectada
	VibrationSeverity *int  `json:"vibration_severity,omitempty"` // 0=none, 1=low, 2=medium, 3=high

	// === QUALIDADE DO AR (OPCIONAL) ===
	Pm2_5Ugm3 *float64 `json:"pm2_5_ugm3,omitempty"` // µg/m³ - partículas finas
	Pm10Ugm3  *float64 `json:"pm10_ugm3,omitempty"`  // µg/m³ - partículas grossas
	CoPpm     *int     `json:"co_ppm,omitempty"`     // ppm - monóxido de carbono
	Ch4Ppm    *int     `json:"ch4_ppm,omitempty"`    // ppm - metano

	// === METADADOS ===
	SourcePoller          string `json:"source_poller"`           // ex: "local_agent", "snmp_poller", "modbus", "script"
	CollectionIntervalSec int    `json:"collection_interval_sec"` //Intervalo em segundos desta medição
	RawData               []byte `json:"raw_data,omitempty"`      // Dados brutos originais (para debug/audit)

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SiteEnvironmentFilter define critérios para filtrar e consultar medições de ambiente.
type SiteEnvironmentFilter struct {
	SiteID            uuid.UUID
	MeasuredAtAfter   *time.Time
	MeasuredAtBefore  *time.Time
	ReceivedAtAfter   *time.Time
	ReceivedAtBefore  *time.Time
	InternalTempCMin  *float64
	InternalTempCMax  *float64
	HumidityPctMin    *float64
	HumidityPctMax    *float64
	DoorOpen          *bool
	MainsPowerOk      *bool
	UpsOnBattery      *bool
	GeneratorRunning  *bool
	SmokeDetected     *bool
	VibrationDetected *bool
	Limit             int
	Offset            int
	OrderBy           []string // ex: []string{"measured_at DESC", "site_id ASC"}
}

// NOTA: SiteEnvironmentRepository NÃO é definida aqui — ver
// internal/core/interfaces/site_environment.go (hexagonal architecture:
// ports ficam em core/interfaces, não em domain).
