// Package database contém implementações de acesso a dados usando SQL e PostgreSQL.
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
)

// siteEnvironmentRepository implementa interfaces.SiteEnvironmentRepository
// usando PostgreSQL.
type siteEnvironmentRepository struct {
	db *sql.DB
}

// NewSiteEnvironmentRepository cria um novo repositório de medições de
// ambiente do site baseado em PostgreSQL.
func NewSiteEnvironmentRepository(db *sql.DB) interfaces.SiteEnvironmentRepository {
	return &siteEnvironmentRepository{db: db}
}

const siteEnvironmentColumns = `
	id, site_id, measured_at, received_at,
	internal_temp_c, temp_risk_point_c, temp_delta_c,
	humidity_pct, humidity_risk_point_pct,
	door_open, door_open_secondary,
	mains_power_ok, ups_on_battery, ups_battery_pct, ups_load_pct,
	generator_running, generator_load_pct, generator_fuel_pct,
	mains_voltage_v, mains_frequency_hz,
	smoke_detected, smoke_detected_zone_2, smoke_detected_zone_3,
	vibration_detected, vibration_severity,
	pm2_5_ugm3, pm10_ugm3, co_ppm, ch4_ppm,
	source_poller, collection_interval_sec, raw_data,
	created_at, updated_at
`

// Create insere uma nova medição de ambiente.
func (r *siteEnvironmentRepository) Create(ctx context.Context, env *domain.SiteEnvironment) error {
	if env == nil {
		return errors.New("environment cannot be nil")
	}

	if env.ID == uuid.Nil {
		env.ID = uuid.New()
	}

	now := time.Now().UTC()
	if env.ReceivedAt.IsZero() {
		env.ReceivedAt = now
	}
	if env.CreatedAt.IsZero() {
		env.CreatedAt = now
	}
	env.UpdatedAt = now

	query := `
		INSERT INTO site_environment (` + siteEnvironmentColumns + `)
		VALUES (
			$1, $2, $3, $4,
			$5, $6, $7,
			$8, $9,
			$10, $11,
			$12, $13, $14, $15,
			$16, $17, $18,
			$19, $20,
			$21, $22, $23,
			$24, $25,
			$26, $27, $28, $29,
			$30, $31, $32,
			$33, $34
		)
	`

	_, err := r.db.ExecContext(
		ctx, query,
		env.ID, env.SiteID, env.MeasuredAt, env.ReceivedAt,
		env.InternalTempC, env.TempRiskPointC, env.TempDeltaC,
		env.HumidityPct, env.HumidityRiskPointPct,
		env.DoorOpen, env.DoorOpenSecondary,
		env.MainsPowerOk, env.UpsOnBattery, env.UpsBatteryPct, env.UpsLoadPct,
		env.GeneratorRunning, env.GeneratorLoadPct, env.GeneratorFuelPct,
		env.MainsVoltageV, env.MainsFrequencyHz,
		env.SmokeDetected, env.SmokeDetectedZone2, env.SmokeDetectedZone3,
		env.VibrationDetected, env.VibrationSeverity,
		env.Pm2_5Ugm3, env.Pm10Ugm3, env.CoPpm, env.Ch4Ppm,
		env.SourcePoller, env.CollectionIntervalSec, env.RawData,
		env.CreatedAt, env.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("falha ao inserir medição de ambiente: %w", err)
	}

	return nil
}

// CreateMany insere múltiplas medições de uma vez, numa única transação.
func (r *siteEnvironmentRepository) CreateMany(ctx context.Context, envs []*domain.SiteEnvironment) error {
	if len(envs) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("falha ao iniciar transação: %w", err)
	}

	query := `
		INSERT INTO site_environment (` + siteEnvironmentColumns + `)
		VALUES (
			$1, $2, $3, $4,
			$5, $6, $7,
			$8, $9,
			$10, $11,
			$12, $13, $14, $15,
			$16, $17, $18,
			$19, $20,
			$21, $22, $23,
			$24, $25,
			$26, $27, $28, $29,
			$30, $31, $32,
			$33, $34
		)
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("falha ao preparar statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now().UTC()

	for _, env := range envs {
		if env == nil {
			continue
		}
		if env.ID == uuid.Nil {
			env.ID = uuid.New()
		}
		if env.ReceivedAt.IsZero() {
			env.ReceivedAt = now
		}
		if env.CreatedAt.IsZero() {
			env.CreatedAt = now
		}
		env.UpdatedAt = now

		if _, err = stmt.ExecContext(
			ctx,
			env.ID, env.SiteID, env.MeasuredAt, env.ReceivedAt,
			env.InternalTempC, env.TempRiskPointC, env.TempDeltaC,
			env.HumidityPct, env.HumidityRiskPointPct,
			env.DoorOpen, env.DoorOpenSecondary,
			env.MainsPowerOk, env.UpsOnBattery, env.UpsBatteryPct, env.UpsLoadPct,
			env.GeneratorRunning, env.GeneratorLoadPct, env.GeneratorFuelPct,
			env.MainsVoltageV, env.MainsFrequencyHz,
			env.SmokeDetected, env.SmokeDetectedZone2, env.SmokeDetectedZone3,
			env.VibrationDetected, env.VibrationSeverity,
			env.Pm2_5Ugm3, env.Pm10Ugm3, env.CoPpm, env.Ch4Ppm,
			env.SourcePoller, env.CollectionIntervalSec, env.RawData,
			env.CreatedAt, env.UpdatedAt,
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("falha ao inserir medição de ambiente (batch): %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("falha ao commitar transação: %w", err)
	}

	return nil
}

// List retorna medições baseado em filtros.
func (r *siteEnvironmentRepository) List(ctx context.Context, filter *domain.SiteEnvironmentFilter) ([]*domain.SiteEnvironment, int, error) {
	if filter == nil {
		filter = &domain.SiteEnvironmentFilter{}
	}

	where := "WHERE 1=1"
	var filterArgs []interface{}
	argIdx := 1

	addCond := func(cond string, val interface{}) {
		where += fmt.Sprintf(" AND %s $%d", cond, argIdx)
		filterArgs = append(filterArgs, val)
		argIdx++
	}

	if filter.SiteID != uuid.Nil {
		addCond("site_id =", filter.SiteID)
	}
	if filter.MeasuredAtAfter != nil {
		addCond("measured_at >=", *filter.MeasuredAtAfter)
	}
	if filter.MeasuredAtBefore != nil {
		addCond("measured_at <=", *filter.MeasuredAtBefore)
	}
	if filter.ReceivedAtAfter != nil {
		addCond("received_at >=", *filter.ReceivedAtAfter)
	}
	if filter.ReceivedAtBefore != nil {
		addCond("received_at <=", *filter.ReceivedAtBefore)
	}
	if filter.InternalTempCMin != nil {
		addCond("internal_temp_c >=", *filter.InternalTempCMin)
	}
	if filter.InternalTempCMax != nil {
		addCond("internal_temp_c <=", *filter.InternalTempCMax)
	}
	if filter.HumidityPctMin != nil {
		addCond("humidity_pct >=", *filter.HumidityPctMin)
	}
	if filter.HumidityPctMax != nil {
		addCond("humidity_pct <=", *filter.HumidityPctMax)
	}
	if filter.DoorOpen != nil {
		addCond("door_open =", *filter.DoorOpen)
	}
	if filter.MainsPowerOk != nil {
		addCond("mains_power_ok =", *filter.MainsPowerOk)
	}
	if filter.UpsOnBattery != nil {
		addCond("ups_on_battery =", *filter.UpsOnBattery)
	}
	if filter.GeneratorRunning != nil {
		addCond("generator_running =", *filter.GeneratorRunning)
	}
	if filter.SmokeDetected != nil {
		addCond("smoke_detected =", *filter.SmokeDetected)
	}
	if filter.VibrationDetected != nil {
		addCond("vibration_detected =", *filter.VibrationDetected)
	}

	orderBy := "ORDER BY measured_at DESC"
	if len(filter.OrderBy) > 0 {
		orderBy = "ORDER BY "
		for i, o := range filter.OrderBy {
			if i > 0 {
				orderBy += ", "
			}
			orderBy += o
		}
	}

	// queryArgs = filterArgs + limit/offset, nesta ordem; countArgs usa
	// só filterArgs, já que COUNT(*) não precisa de paginação.
	queryArgs := append([]interface{}{}, filterArgs...)

	limitOffset := ""
	if filter.Limit > 0 {
		limitOffset += fmt.Sprintf(" LIMIT $%d", argIdx)
		queryArgs = append(queryArgs, filter.Limit)
		argIdx++
	}
	if filter.Offset >= 0 {
		limitOffset += fmt.Sprintf(" OFFSET $%d", argIdx)
		queryArgs = append(queryArgs, filter.Offset)
		argIdx++
	}

	query := "SELECT " + siteEnvironmentColumns + " FROM site_environment " + where + " " + orderBy + limitOffset

	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("falha ao executar query de ambiente: %w", err)
	}
	defer rows.Close()

	var envs []*domain.SiteEnvironment
	for rows.Next() {
		env := &domain.SiteEnvironment{}
		if err := scanSiteEnvironment(rows, env); err != nil {
			return nil, 0, fmt.Errorf("falha ao escanear medição de ambiente: %w", err)
		}
		envs = append(envs, env)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("erro ao iterar resultados de ambiente: %w", err)
	}

	countQuery := "SELECT COUNT(*) FROM site_environment " + where
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, filterArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("falha ao contar medições de ambiente: %w", err)
	}

	return envs, total, nil
}

// GetLatest retorna a medição mais recente para um site.
func (r *siteEnvironmentRepository) GetLatest(ctx context.Context, siteID uuid.UUID) (*domain.SiteEnvironment, error) {
	query := "SELECT " + siteEnvironmentColumns + ` FROM site_environment WHERE site_id = $1 ORDER BY measured_at DESC LIMIT 1`

	row := r.db.QueryRowContext(ctx, query, siteID)

	env := &domain.SiteEnvironment{}
	if err := scanSiteEnvironment(row, env); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("falha ao buscar medição mais recente de ambiente: %w", err)
	}

	return env, nil
}

// DeleteOlderThan remove medições mais antigas que um determinado tempo.
func (r *siteEnvironmentRepository) DeleteOlderThan(ctx context.Context, olderThan time.Time) error {
	if olderThan.IsZero() {
		return errors.New("olderThan timestamp não pode ser zero")
	}

	_, err := r.db.ExecContext(ctx, `DELETE FROM site_environment WHERE received_at < $1`, olderThan)
	if err != nil {
		return fmt.Errorf("falha ao excluir medições antigas de ambiente: %w", err)
	}

	return nil
}

// rowScanner abstrai *sql.Row e *sql.Rows, que partilham o método Scan
// mas não têm uma interface comum na stdlib.
type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanSiteEnvironment(row rowScanner, env *domain.SiteEnvironment) error {
	return row.Scan(
		&env.ID, &env.SiteID, &env.MeasuredAt, &env.ReceivedAt,
		&env.InternalTempC, &env.TempRiskPointC, &env.TempDeltaC,
		&env.HumidityPct, &env.HumidityRiskPointPct,
		&env.DoorOpen, &env.DoorOpenSecondary,
		&env.MainsPowerOk, &env.UpsOnBattery, &env.UpsBatteryPct, &env.UpsLoadPct,
		&env.GeneratorRunning, &env.GeneratorLoadPct, &env.GeneratorFuelPct,
		&env.MainsVoltageV, &env.MainsFrequencyHz,
		&env.SmokeDetected, &env.SmokeDetectedZone2, &env.SmokeDetectedZone3,
		&env.VibrationDetected, &env.VibrationSeverity,
		&env.Pm2_5Ugm3, &env.Pm10Ugm3, &env.CoPpm, &env.Ch4Ppm,
		&env.SourcePoller, &env.CollectionIntervalSec, &env.RawData,
		&env.CreatedAt, &env.UpdatedAt,
	)
}