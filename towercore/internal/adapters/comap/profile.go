// Package comap implementa o adapter para controladores de grupo gerador
// ComAp (ex.: InteliLite/InteliGen), lidos via Modbus (RTU ou TCP).
//
// IMPORTANTE — estado de validação:
// Apenas 4 registos foram confirmados por engenharia reversa (dump de memória
// vs. leitura no WebSupervisor). Os restantes campos de telemetria de energia
// do site (SOH de bateria, % de carga, backhaul, células/PRB) NÃO se originam
// neste controlador — ver docs/definições.md. Este profile não deve ser
// expandido com registos não validados empiricamente em campo.
package comap

import (
	"context"
	"fmt"
	"time"

	"towercore/internal/core/interfaces"
)

// RegisterAddress identifica o endereço Modbus de um registo holding.
type RegisterAddress uint16

// Endereços de registo validados por engenharia reversa (dump vs. WebSupervisor).
// Ver docs/definições.md para a matriz de credibilidade completa.
const (
	RegFuelPercent    RegisterAddress = 54  // Confiança: 85% — pendente validação em reabastecimento
	RegFuelLiters     RegisterAddress = 55  // Confiança: 100% — leitura direta 1:1
	RegBatteryVoltage RegisterAddress = 50  // Confiança: 100%, fator de escala sob verificação em campo
	RegRunHours       RegisterAddress = 152 // Confiança: 100% — acumulado, validado por delta temporal
)

// ValidationStatus descreve o grau de confiança de um campo, herdado do
// relatório de engenharia reversa. Usado para não tratar dados 85%-confiáveis
// como se fossem equivalentes aos 100%-confirmados.
type ValidationStatus string

const (
	StatusValidated   ValidationStatus = "validated"   // confirmado empiricamente em campo
	StatusUnconfirmed ValidationStatus = "unconfirmed" // coerência matemática, sem validação direta
)

// RegisterDef descreve como interpretar um registo holding: fator de escala
// e status de validação. O fator de escala do Registo 50 (tensão de bateria)
// está marcado como não confirmado — o relatório de origem apresentava uma
// inconsistência (12.6V vs 12.5V) que ainda não foi esclarecida em campo.
type RegisterDef struct {
	Address RegisterAddress
	Name    string
	Scale   float64
	Status  ValidationStatus
}

// Profile é o mapeamento de registos Modbus do ComAp usado pelo adapter.
var Profile = map[string]RegisterDef{
	"fuel_liters": {
		Address: RegFuelLiters,
		Name:    "fuel_liters",
		Scale:   1.0,
		Status:  StatusValidated,
	},
	"fuel_percent": {
		Address: RegFuelPercent,
		Name:    "fuel_percent",
		Scale:   1.0,
		Status:  StatusUnconfirmed,
	},
	"battery_voltage": {
		Address: RegBatteryVoltage,
		Name:    "battery_voltage",
		Scale:   0.1, // fator sob verificação — confirmar antes de assumir como definitivo
		Status:  StatusUnconfirmed,
	},
	"run_hours": {
		Address: RegRunHours,
		Name:    "run_hours",
		Scale:   1.0,
		Status:  StatusValidated,
	},
}

// Metrics representa a telemetria de energia lida de um controlador ComAp
// para uma torre. Campos usam ponteiro para nunca fabricar valor: um registo
// que falhou a leitura ou não foi confirmado em campo deve chegar como nil
// ao consumidor, nunca como zero implícito (princípio "no fabricated data").
type Metrics struct {
	FuelLiters      *float64
	FuelPercent     *float64
	BatteryVoltageV *float64
	RunHoursTotal   *float64
	CollectedAt     time.Time
}

// Reader lê o Profile de um controlador ComAp através de um
// interfaces.ModbusClient (port definido em core/interfaces).
type Reader struct {
	client  interfaces.ModbusClient
	slaveID byte
}

// NewReader cria um leitor de telemetria ComAp.
func NewReader(client interfaces.ModbusClient, slaveID byte) *Reader {
	return &Reader{client: client, slaveID: slaveID}
}

// Read consulta os 4 registos validados do Profile e devolve Metrics.
// Falhas de leitura individuais não abortam a coleta inteira: o campo fica
// nil e o erro é agregado, seguindo o mesmo padrão de tolerância a falhas
// parciais usado nos adapters SNMP existentes.
func (r *Reader) Read(ctx context.Context) (*Metrics, error) {
	m := &Metrics{CollectedAt: time.Now().UTC()}
	var errs []error

	if v, err := r.readScaled(Profile["fuel_liters"]); err != nil {
		errs = append(errs, fmt.Errorf("fuel_liters: %w", err))
	} else {
		m.FuelLiters = v
	}

	if v, err := r.readScaled(Profile["fuel_percent"]); err != nil {
		errs = append(errs, fmt.Errorf("fuel_percent: %w", err))
	} else {
		m.FuelPercent = v
	}

	if v, err := r.readScaled(Profile["battery_voltage"]); err != nil {
		errs = append(errs, fmt.Errorf("battery_voltage: %w", err))
	} else {
		m.BatteryVoltageV = v
	}

	if v, err := r.readScaled(Profile["run_hours"]); err != nil {
		errs = append(errs, fmt.Errorf("run_hours: %w", err))
	} else {
		m.RunHoursTotal = v
	}

	if len(errs) > 0 {
		return m, fmt.Errorf("comap: %d/4 registos falharam: %v", len(errs), errs)
	}
	return m, nil
}

// readScaled lê um único registo holding (16 bits, unsigned) e aplica o
// fator de escala definido no Profile.
func (r *Reader) readScaled(def RegisterDef) (*float64, error) {
	raw, err := r.client.ReadHoldingRegisters(uint16(def.Address), 1)
	if err != nil {
		return nil, err
	}
	if len(raw) < 2 {
		return nil, fmt.Errorf("resposta curta para registo %d (%s)", def.Address, def.Name)
	}
	rawValue := float64(uint16(raw[0])<<8 | uint16(raw[1]))
	scaled := rawValue * def.Scale
	return &scaled, nil
}