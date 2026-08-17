// Package comap implementa o adapter para controladores de grupo gerador
// ComAp (ex.: InteliLite/InteliGen), lidos via Modbus (RTU ou TCP).
//
// IMPORTANTE — estado de validação:
// Registos confirmados em campo via comapcheck (Ago 2026):
//
//	50 = Battery Voltage (×0.1 V)
//	54 = Engine Temp (°C)
//	53 / 55 = frequentemente 0x8000 (sensor não configurado)
//
// RegRunHours (152) mantido do mapeamento anterior — validar em campo.
package comap

import (
	"context"
	"errors"
	"fmt"
	"time"

	"towercore/internal/core/interfaces"
)

// RegisterAddress identifica o endereço Modbus de um registo holding.
type RegisterAddress uint16

const (
	RegBatteryVoltage RegisterAddress = 50  // ×0.1 V
	RegOilPressure    RegisterAddress = 53  // ×0.1 bar (muitas vezes 0x8000)
	RegEngineTemp     RegisterAddress = 54  // °C
	RegFuelLevel      RegisterAddress = 55  // % (muitas vezes 0x8000)
	RegRunHours       RegisterAddress = 152 // validar em campo

	// === Elétrica do Gerador ===
	RegFrequency   RegisterAddress = 116 // ×0.01 Hz
	RegCurrentL1   RegisterAddress = 110 // ×0.1 A
	RegCurrentL2   RegisterAddress = 111 // ×0.1 A
	RegCurrentL3   RegisterAddress = 112 // ×0.1 A
	RegVoltageL1L2 RegisterAddress = 100 // ×0.1 V (opcional)
	RegVoltageL2L3 RegisterAddress = 101 // ×0.1 V (opcional)
	RegVoltageL3L1 RegisterAddress = 102 // ×0.1 V (opcional)
)

// ErrInactiveRegister indica que o registo existe mas está marcado como
// inativo/desconfigurado no controlador (valor 0x8000). Não é falha de rede.
var ErrInactiveRegister = errors.New("registo inativo/desconfigurado")

// ValidationStatus descreve o grau de confiança de um campo.
type ValidationStatus string

const (
	StatusValidated   ValidationStatus = "validated"
	StatusUnconfirmed ValidationStatus = "unconfirmed"
)

// RegisterDef descreve como interpretar um registo holding.
type RegisterDef struct {
	Address RegisterAddress
	Name    string
	Scale   float64
	Signed  bool // true para engine_temp (pode ser negativo)
	Status  ValidationStatus
}

// Profile é o mapeamento de registos Modbus do ComAp usado pelo adapter.
var Profile = map[string]RegisterDef{
	"battery_voltage": {
		Address: RegBatteryVoltage,
		Name:    "battery_voltage",
		Scale:   0.1,
		Signed:  false,
		Status:  StatusValidated,
	},
	"engine_temp": {
		Address: RegEngineTemp,
		Name:    "engine_temp",
		Scale:   1.0,
		Signed:  true, // int16 — pode ser negativo
		Status:  StatusValidated,
	},
	"fuel_level": {
		Address: RegFuelLevel,
		Name:    "fuel_level",
		Scale:   1.0,
		Signed:  false,
		Status:  StatusUnconfirmed,
	},
	"oil_pressure": {
		Address: RegOilPressure,
		Name:    "oil_pressure",
		Scale:   0.1,
		Signed:  true,
		Status:  StatusUnconfirmed,
	},
	"run_hours": {
		Address: RegRunHours,
		Name:    "run_hours",
		Scale:   1.0,
		Signed:  false,
		Status:  StatusValidated,
	},

	// === Métricas Elétricas do Gerador ===
	"frequency_hz": {
		Address: RegFrequency,
		Name:    "frequency_hz",
		Scale:   0.01, // ×0.01 Hz → Hz
		Signed:  false,
		Status:  StatusValidated, // Geralmente confiável
	},
	"current_l1_a": {
		Address: RegCurrentL1,
		Name:    "current_l1_a",
		Scale:   0.1, // ×0.1 A → A
		Signed:  false,
		Status:  StatusValidated,
	},
	"current_l2_a": {
		Address: RegCurrentL2,
		Name:    "current_l2_a",
		Scale:   0.1, // ×0.1 A → A
		Signed:  false,
		Status:  StatusValidated,
	},
	"current_l3_a": {
		Address: RegCurrentL3,
		Name:    "current_l3_a",
		Scale:   0.1, // ×0.1 A → A
		Signed:  false,
		Status:  StatusValidated,
	},

	// FIX: estas 3 chaves estavam ausentes do Profile. Read() já as chamava
	// (voltage_l1_l2_v/l2_l3_v/l3_l1_v), então caíam no zero-value do map
	// (Address=0, Scale=0) e devolviam sempre 0.0 sem erro — dado fabricado
	// silenciosamente, violando a invariante de nil documentada acima.
	"voltage_l1_l2_v": {
		Address: RegVoltageL1L2,
		Name:    "voltage_l1_l2_v",
		Scale:   0.1, // ×0.1 V → V
		Signed:  false,
		Status:  StatusUnconfirmed, // ainda não validado em campo
	},
	"voltage_l2_l3_v": {
		Address: RegVoltageL2L3,
		Name:    "voltage_l2_l3_v",
		Scale:   0.1,
		Signed:  false,
		Status:  StatusUnconfirmed,
	},
	"voltage_l3_l1_v": {
		Address: RegVoltageL3L1,
		Name:    "voltage_l3_l1_v",
		Scale:   0.1,
		Signed:  false,
		Status:  StatusUnconfirmed,
	},
}

// Metrics representa a telemetria lida de um controlador ComAp.
// Ponteiros: nil = registo falhou ou está inativo (nunca fabricar zero).
type Metrics struct {
	BatteryVoltageV *float64
	EngineTempC     *float64
	FuelLevelPct    *float64
	OilPressureBar  *float64
	RunHoursTotal   *float64

	// === Elétrica do Gerador ===
	FrequencyHz  *float64
	CurrentL1A   *float64
	CurrentL2A   *float64
	CurrentL3A   *float64
	VoltageL1L2V *float64
	VoltageL2L3V *float64
	VoltageL3L1V *float64

	CollectedAt time.Time
}

// Reader lê o Profile de um controlador ComAp.
type Reader struct {
	client  interfaces.ModbusClient
	slaveID byte
}

// NewReader cria um leitor de telemetria ComAp.
func NewReader(client interfaces.ModbusClient, slaveID byte) *Reader {
	return &Reader{client: client, slaveID: slaveID}
}

// Read consulta os registos do Profile.
// Falhas individuais não abortam a coleta: o campo fica nil.
// Só devolve erro se NENHUM registo útil foi lido.
func (r *Reader) Read(ctx context.Context) (*Metrics, error) {
	m := &Metrics{CollectedAt: time.Now().UTC()}
	var errs []error
	ok := 0

	read := func(key string, dest **float64) {
		v, err := r.readScaled(Profile[key])
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", key, err))
			return
		}
		if v != nil {
			*dest = v
			ok++
		}
		// v == nil → registo inativo (0x8000), campo fica nil sem erro
	}

	read("battery_voltage", &m.BatteryVoltageV)
	read("engine_temp", &m.EngineTempC)
	read("fuel_level", &m.FuelLevelPct)
	read("oil_pressure", &m.OilPressureBar)
	read("run_hours", &m.RunHoursTotal)
	read("frequency_hz", &m.FrequencyHz)
	read("current_l1_a", &m.CurrentL1A)
	read("current_l2_a", &m.CurrentL2A)
	read("current_l3_a", &m.CurrentL3A)
	read("voltage_l1_l2_v", &m.VoltageL1L2V)
	read("voltage_l2_l3_v", &m.VoltageL2L3V)
	read("voltage_l3_l1_v", &m.VoltageL3L1V)

	if ok == 0 && len(errs) > 0 {
		return m, fmt.Errorf("comap: nenhum registo útil (%d falhas): %v", len(errs), errs)
	}
	if len(errs) > 0 {
		// partial success — devolve métricas + erro informativo
		return m, fmt.Errorf("comap: %d registos falharam (parciais ok): %v", len(errs), errs)
	}
	return m, nil
}

// readScaled lê um registo holding e aplica escala.
// Retorna (nil, nil) se o registo estiver inativo (0x8000).
func (r *Reader) readScaled(def RegisterDef) (*float64, error) {
	raw, err := r.client.ReadHoldingRegisters(uint16(def.Address), 1)
	if err != nil {
		if errors.Is(err, ErrInactiveRegister) {
			return nil, nil
		}
		return nil, err
	}
	if len(raw) < 2 {
		return nil, fmt.Errorf("resposta curta para registo %d (%s)", def.Address, def.Name)
	}

	var rawValue float64
	if def.Signed {
		rawValue = float64(int16(raw[0])<<8 | int16(raw[1]))
	} else {
		rawValue = float64(uint16(raw[0])<<8 | uint16(raw[1]))
	}
	scaled := rawValue * def.Scale
	return &scaled, nil
}