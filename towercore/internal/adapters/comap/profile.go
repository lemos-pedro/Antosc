// Package comap implementa o adapter para controladores de grupo gerador
// ComAp (ex.: InteliLite/InteliGen), lidos via Modbus (RTU ou TCP).
//
// IMPORTANTE — estado de validação:
// Registos confirmados em campo via comapcheck (Ago 2026):
//   50 = Battery Voltage (×0.1 V)
//   54 = Engine Temp (°C)
//   53 / 55 = frequentemente 0x8000 (sensor não configurado)
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
}

// Metrics representa a telemetria lida de um controlador ComAp.
// Ponteiros: nil = registo falhou ou está inativo (nunca fabricar zero).
type Metrics struct {
	BatteryVoltageV *float64
	EngineTempC     *float64
	FuelLevelPct    *float64
	OilPressureBar  *float64
	RunHoursTotal   *float64
	CollectedAt     time.Time
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