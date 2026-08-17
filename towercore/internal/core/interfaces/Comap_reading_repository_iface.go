package interfaces

import (
	"context"
	"errors"
	"time"
)

// ComapReading é a última leitura de telemetria ComAp conhecida para uma
// torre. Campos ponteiro: nil significa "não lido/não confirmado", nunca
// zero fabricado — consistente com o princípio "no fabricated data".
type ComapReading struct {
	TowerID         string
	FuelLiters      *float64
	FuelPercent     *float64 // StatusUnconfirmed — frontend deve marcar como não validado
	BatteryVoltageV *float64 // StatusUnconfirmed — frontend deve marcar como não validado
	RunHoursTotal   *float64

	// === Elétrica do Gerador (migration 00008) ===
	FrequencyHz  *float64
	CurrentL1A   *float64
	CurrentL2A   *float64
	CurrentL3A   *float64
	VoltageL1L2V *float64 // StatusUnconfirmed — frontend deve marcar como não validado
	VoltageL2L3V *float64 // StatusUnconfirmed — frontend deve marcar como não validado
	VoltageL3L1V *float64 // StatusUnconfirmed — frontend deve marcar como não validado

	CollectedAt *time.Time
}

// ComapReadingRepository persiste a última leitura ComAp por torre.
type ComapReadingRepository interface {
	Upsert(ctx context.Context, r *ComapReading) error
	GetByTowerID(ctx context.Context, towerID string) (*ComapReading, error)
}

// ErrComapReadingNotFound segue o mesmo padrão de ErrTowerNotFound /
// ErrEventNotFound / ErrTicketNotFound já existentes no pacote.
var ErrComapReadingNotFound = errors.New("comap reading not found")