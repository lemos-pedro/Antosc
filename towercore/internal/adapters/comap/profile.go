// Package comap implementa o adapter para controladores de grupo gerador
// ComAp (ex.: InteliLite/InteliGen), lidos via Modbus TCP.
//
// IMPORTANTE — estado de validação (30 Set 2026):
// - Confirmado em campo: BatteryVoltage(50), EngineTemp(54), Frequency(11),
//   RunHours(139)
// - Forte (padrão matemático bate, não fotografado no ecrã): RPM(10)
// - Confirmado por endereço, fonte (Mains/Gen) em aberto: Voltage L1-3N(0-2)
// - Sensor não instalado no controlador (0x8000 constante): OilPressure(53),
//   FuelLevel(55), LoadKW/EnergyKWh (bloco 186-219 quase todo 0x8000) —
//   decisão: mostrar "—" no dashboard, nunca calcular nem mostrar zero
// - Regs 206/207 (1500/500) parecem potência nominal de placa (estático,
//   fora do polling dinâmico)
//
// Ver /learnings-and-conventions.md para o histórico de validação.
package comap

import (
	"context"
	"errors"
	"fmt"
	"time"

	"towercore/internal/core/interfaces"
)

type RegisterAddress uint16

const (
	// === Confirmado ===
	RegBatteryVoltage RegisterAddress = 50  // ×0.1 V
	RegEngineTemp     RegisterAddress = 54  // °C
	RegRunHours       RegisterAddress = 139 // horas totais
	RegFrequency      RegisterAddress = 11  // ×0.1 Hz
	RegRPM            RegisterAddress = 10  // rotações — forte

	// L1/L2/L3 — endereço confirmado, fonte (Mains ou Gen) em aberto
	RegVoltageL1N RegisterAddress = 0
	RegVoltageL2N RegisterAddress = 1
	RegVoltageL3N RegisterAddress = 2

	// === Sem sensor instalado (confirmado 0x8000 persistente) ===
	RegOilPressure RegisterAddress = 53
	RegFuelLevel   RegisterAddress = 55

	// === Placa nominal, estático — não fazer polling dinâmico ===
	RegRatedPowerKVA RegisterAddress = 206

	// === Bloco de alarmes ===
	RegAlarmBlockStart RegisterAddress = 90
	RegAlarmBlockCount uint16          = 49 // 90-138
)

var ErrInactiveRegister = errors.New("registo inativo/desconfigurado (0x8000)")

type ValidationStatus string

const (
	StatusConfirmed        ValidationStatus = "confirmed"
	StatusStrong           ValidationStatus = "strong"
	StatusUnconfirmed       ValidationStatus = "unconfirmed"
	StatusSensorUnavailable ValidationStatus = "sensor_unavailable"
)

type RegisterDef struct {
	Address RegisterAddress
	Name    string
	Scale   float64
	Signed  bool
	Status  ValidationStatus
}

// Profile é o mapeamento de registos Modbus do ComAp usado pelo adapter.
// Campos com Status == StatusSensorUnavailable não são lidos (ver Read).
var Profile = map[string]RegisterDef{
	"battery_voltage_v": {Address: RegBatteryVoltage, Name: "battery_voltage_v", Scale: 0.1, Signed: false, Status: StatusConfirmed},
	"engine_temp_c":     {Address: RegEngineTemp, Name: "engine_temp_c", Scale: 1.0, Signed: true, Status: StatusConfirmed},
	"run_hours_total":   {Address: RegRunHours, Name: "run_hours_total", Scale: 1.0, Signed: false, Status: StatusConfirmed},
	"frequency_hz":      {Address: RegFrequency, Name: "frequency_hz", Scale: 0.1, Signed: false, Status: StatusConfirmed},
	"rpm":               {Address: RegRPM, Name: "rpm", Scale: 1.0, Signed: false, Status: StatusStrong},

	"voltage_l1n_v": {Address: RegVoltageL1N, Name: "voltage_l1n_v", Scale: 1.0, Signed: false, Status: StatusUnconfirmed},
	"voltage_l2n_v": {Address: RegVoltageL2N, Name: "voltage_l2n_v", Scale: 1.0, Signed: false, Status: StatusUnconfirmed},
	"voltage_l3n_v": {Address: RegVoltageL3N, Name: "voltage_l3n_v", Scale: 1.0, Signed: false, Status: StatusUnconfirmed},

	"oil_pressure_bar": {Address: RegOilPressure, Name: "oil_pressure_bar", Scale: 0.1, Signed: true, Status: StatusSensorUnavailable},
	"fuel_level_pct":   {Address: RegFuelLevel, Name: "fuel_level_pct", Scale: 0.1, Signed: false, Status: StatusSensorUnavailable},
}

// Metrics representa a telemetria lida de um controlador ComAp.
// Ponteiros: nil = registo falhou, inativo, ou sem sensor.
type Metrics struct {
	BatteryVoltageV *float64
	EngineTempC     *float64
	RunHoursTotal   *float64
	FrequencyHz     *float64
	RPM             *float64

	VoltageL1NV *float64
	VoltageL2NV *float64
	VoltageL3NV *float64

	OilPressureBar *float64 // sempre nil — sensor não instalado
	FuelLevelPct   *float64 // sempre nil — sensor não instalado

	AlarmsActive []int // índices dos registos 90-138 com valor != 0

	CollectedAt time.Time
}

type Reader struct {
	client  interfaces.ModbusClient
	slaveID byte
}

func NewReader(client interfaces.ModbusClient, slaveID byte) *Reader {
	return &Reader{client: client, slaveID: slaveID}
}

// Read consulta os registos do Profile.
// Campos StatusSensorUnavailable não são lidos (poupa chamadas Modbus).
// Falhas individuais não abortam a coleta: o campo fica nil.
func (r *Reader) Read(ctx context.Context) (*Metrics, error) {
	m := &Metrics{CollectedAt: time.Now().UTC()}
	var errs []error
	ok := 0

	read := func(key string, dest **float64) {
		def := Profile[key]
		if def.Status == StatusSensorUnavailable {
			return
		}
		v, err := r.readScaled(def)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", key, err))
			return
		}
		if v != nil {
			*dest = v
			ok++
		}
	}

	read("battery_voltage_v", &m.BatteryVoltageV)
	read("engine_temp_c", &m.EngineTempC)
	read("run_hours_total", &m.RunHoursTotal)
	read("frequency_hz", &m.FrequencyHz)
	read("rpm", &m.RPM)
	read("voltage_l1n_v", &m.VoltageL1NV)
	read("voltage_l2n_v", &m.VoltageL2NV)
	read("voltage_l3n_v", &m.VoltageL3NV)

	alarmRaw, err := r.readAlarmBlock()
	if err != nil {
		errs = append(errs, fmt.Errorf("alarm_block: %w", err))
	} else {
		m.AlarmsActive = alarmRaw
		ok++
	}

	if ok == 0 && len(errs) > 0 {
		return m, fmt.Errorf("comap: nenhum registo útil (%d falhas): %v", len(errs), errs)
	}
	if len(errs) > 0 {
		return m, fmt.Errorf("comap: %d falhas parciais: %v", len(errs), errs)
	}
	return m, nil
}

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
	v := uint16(raw[0])<<8 | uint16(raw[1])

	var rawValue float64
	if def.Signed {
		rawValue = float64(int16(v))
	} else {
		rawValue = float64(v)
	}
	scaled := rawValue * def.Scale
	return &scaled, nil
}

// readAlarmBlock lê registo a registo (o client atual só suporta quantity=1)
// e devolve os índices com valor ativo. Trocar por leitura em bloco quando
// o TCPClient suportar quantity > 1 — ver TODO no client.
func (r *Reader) readAlarmBlock() ([]int, error) {
	var active []int
	for i := uint16(0); i < RegAlarmBlockCount; i++ {
		addr := uint16(RegAlarmBlockStart) + i
		raw, err := r.client.ReadHoldingRegisters(addr, 1)
		if err != nil {
			if errors.Is(err, ErrInactiveRegister) {
				continue
			}
			return active, err
		}
		v := uint16(raw[0])<<8 | uint16(raw[1])
		if v != 0 {
			active = append(active, int(addr))
		}
	}
	return active, nil
}