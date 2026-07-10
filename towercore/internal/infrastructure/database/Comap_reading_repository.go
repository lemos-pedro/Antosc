package database

import (
	"context"
	"database/sql"
	"errors"

	"towercore/internal/core/interfaces"
)

// ComapReadingRepository persiste a última leitura ComAp por torre em
// comap_readings (migration 00007). Uma linha por torre, upsert a cada
// ciclo do ComapScheduler.
type ComapReadingRepository struct {
	db *sql.DB
}

func NewComapReadingRepository(db *sql.DB) *ComapReadingRepository {
	return &ComapReadingRepository{db: db}
}

func (r *ComapReadingRepository) Upsert(ctx context.Context, reading *interfaces.ComapReading) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const query = `
INSERT INTO comap_readings (
	tower_id, fuel_liters, fuel_percent, battery_voltage_v, run_hours_total,
	collected_at, updated_at
) VALUES (
	$1::uuid, $2, $3, $4, $5, $6, now()
)
ON CONFLICT (tower_id) DO UPDATE SET
	fuel_liters = EXCLUDED.fuel_liters,
	fuel_percent = EXCLUDED.fuel_percent,
	battery_voltage_v = EXCLUDED.battery_voltage_v,
	run_hours_total = EXCLUDED.run_hours_total,
	collected_at = EXCLUDED.collected_at,
	updated_at = now()`

	_, err := r.db.ExecContext(ctx, query,
		reading.TowerID, reading.FuelLiters, reading.FuelPercent,
		reading.BatteryVoltageV, reading.RunHoursTotal, reading.CollectedAt,
	)
	return err
}

func (r *ComapReadingRepository) GetByTowerID(ctx context.Context, towerID string) (*interfaces.ComapReading, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const query = `
SELECT tower_id::text, fuel_liters, fuel_percent, battery_voltage_v,
       run_hours_total, collected_at
FROM comap_readings
WHERE tower_id::text = $1`

	var reading interfaces.ComapReading
	err := r.db.QueryRowContext(ctx, query, towerID).Scan(
		&reading.TowerID, &reading.FuelLiters, &reading.FuelPercent,
		&reading.BatteryVoltageV, &reading.RunHoursTotal, &reading.CollectedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, interfaces.ErrComapReadingNotFound
		}
		return nil, err
	}
	return &reading, nil
}