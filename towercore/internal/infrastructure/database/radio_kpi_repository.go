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
)

// radioKPIRepository implementa a interface RadioKPIGenericRepository usando PostgreSQL.
type radioKPIRepository struct {
	db *sql.DB
}

// NewRadioKPIRepository cria um novo repositório de KPIs de rádio baseado em PostgreSQL.
func NewRadioKPIRepository(db *sql.DB) domain.RadioKPIGenericRepository {
	return &radioKPIRepository{db: db}
}

// CreateMany insere múltiplos KPIs de rádio de uma vez.
func (r *radioKPIRepository) CreateMany(ctx context.Context, kpis []*domain.RadioKPI) error {
	if len(kpis) == 0 {
		return nil
	}

	// Preparar o comando INSERT
	query := `
		INSERT INTO radio_kpi_staging (
			id, tower_id, sector_id, cell_technique,
			measured_at, received_at,
			tx_power_watt, tx_power_dbm, rx_power_dbm, snr_db, sinr_db, rsrp_dbm, rsrq_db,
			connected_ues, max_supported_ues, prb_utilization_pct, channel_occupancy_pct,
			ber, bler, fer, codec_drop_pct,
			ho_attempt, ho_success, ho_fail, ho_ping_pong,
			call_drop_pct, call_block_pct, pdcp_sdu_loss_pct, rlc_retrans_pct,
			source_system, collection_interval_sec, raw_data,
			created_at, updated_at
		) VALUES (
			$id, $id, $id, $id,
			$id, $id,
			$id, $id, $id, $id, $id, $id, $id,
			$id, $id, $id, $id,
			$id, $id, $id, $id,
			$id, $id, $id, $id,
			$id, $id,
			$id, $id,
			$id, $id
		)
	`

	// Usar transação para melhor performance em batch
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("falha ao iniciar transação: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback() // Ignorar erro de rollback se já houver erro
		} else {
			if errTx := tx.Commit(); errTx != nil {
				err = fmt.Errorf("falha ao commitar transação: %w", errTx)
			}
		}
	}()

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("falha ao preparar statement: %w", err)
	}
	defer stmt.Close()

	// Processar cada KPI
	for _, kpi := range kpis {
		// Gerar ID se não fornecido
		if kpi.ID == uuid.Nil {
			kpi.ID = uuid.New()
		}

		// Definir timestamps se não fornecidos
		now := time.Now().UTC()
		if kpi.MeasuredAt.IsZero() {
			kpi.MeasuredAt = now
		}
		if kpi.ReceivedAt.IsZero() {
			kpi.ReceivedAt = now
		}

		// Executar o insert
		_, err = stmt.ExecContext(
			ctx,
			kpi.ID, kpi.TowerID, kpi.SectorID, kpi.CellTechnique,
			kpi.MeasuredAt, kpi.ReceivedAt,
			kpi.TxPowerWatt, kpi.TxPowerDbm, kpi.RxPowerDbm, kpi.SnrDb, kpi.SinrDb, kpi.RsrpDbm, kpi.RsrqDbm,
			kpi.ConnectedUEs, kpi.MaxSupportedUEs, kpi.PRBUtilizationPct, kpi.ChannelOccupancyPct,
			kpi.Ber, kpi.Bler, kpi.Fer, kpi.CodecDropPct,
			kpi.HoAttempt, kpi.HoSuccess, kpi.HoFail, kpi.HoPingPong,
			kpi.CallDropPct, kpi.CallBlockPct, kpi.PDCPSDULossPct, kpi.RLCRetransPct,
			kpi.SourceSystem, kpi.CollectionIntervalSec, kpi.RawData,
			kpi.CreatedAt, kpi.UpdatedAt,
		)

		if err != nil {
			return fmt.Errorf("falha ao inserir RadioKPI: %w", err)
		}
	}

	return nil
}

// List retorna KPIs de rádio baseado em filtros.
func (r *radioKPIRepository) List(ctx context.Context, filter *domain.RadioKPIFilter) ([]*domain.RadioKPI, int, error) {
	if filter == nil {
		filter = &domain.RadioKPIFilter{}
	}

	// Construir a query dinamicamente baseado nos filtros fornecidos
	query := `
		SELECT
			id, tower_id, sector_id, cell_technique,
			measured_at, received_at,
			tx_power_watt, tx_power_dbm, rx_power_dbm, snr_db, sinr_db, rsrp_dbm, rsrq_db,
			connected_ues, max_supported_ues, prb_utilization_pct, channel_occupancy_pct,
			ber, bler, fer, codec_drop_pct,
			ho_attempt, ho_success, ho_fail, ho_ping_pong,
			call_drop_pct, call_block_pct, pdcp_sdu_loss_pct, rlc_retrans_pct,
			source_system, collection_interval_sec, raw_data,
			created_at, updated_at
		FROM radio_kpi_staging
		WHERE 1=1
	`

	var args []interface{}
	argIndex := 1

	// Aplicar filtros
	if filter.TowerID != uuid.Nil {
		query += fmt.Sprintf(" AND tower_id = $%d", argIndex)
		args = append(args, filter.TowerID)
		argIndex++
	}
	if filter.SectorID != nil {
		query += fmt.Sprintf(" AND sector_id = $%d", argIndex)
		args = append(args, *filter.SectorID)
		argIndex++
	}
	if filter.CellTechnique != nil {
		query += fmt.Sprintf(" AND cell_technique = $%d", argIndex)
		args = append(args, *filter.CellTechnique)
		argIndex++
	}
	if filter.MeasuredAtAfter != nil {
		query += fmt.Sprintf(" AND measured_at >= $%d", argIndex)
		args = append(args, *filter.MeasuredAtAfter)
		argIndex++
	}
	if filter.MeasuredAtBefore != nil {
		query += fmt.Sprintf(" AND measured_at <= $%d", argIndex)
		args = append(args, *filter.MeasuredAtBefore)
		argIndex++
	}
	if filter.ReceivedAtAfter != nil {
		query += fmt.Sprintf(" AND received_at >= $%d", argIndex)
		args = append(args, *filter.ReceivedAtAfter)
		argIndex++
	}
	if filter.ReceivedAtBefore != nil {
		query += fmt.Sprintf(" AND received_at <= $%d", argIndex)
		args = append(args, *filter.ReceivedAtBefore)
		argIndex++
	}
	if filter.MinRxPowerDbm != nil {
		query += fmt.Sprintf(" AND rx_power_dbm >= $%d", argIndex)
		args = append(args, *filter.MinRxPowerDbm)
		argIndex++
	}
	if filter.MaxRxPowerDbm != nil {
		query += fmt.Sprintf(" AND rx_power_dbm <= $%d", argIndex)
		args = append(args, *filter.MaxRxPowerDbm)
		argIndex++
	}
	if filter.MinSnrDb != nil {
		query += fmt.Sprintf(" AND snr_db >= $%d", argIndex)
		args = append(args, *filter.MinSnrDb)
		argIndex++
	}
	if filter.MaxSnrDb != nil {
		query += fmt.Sprintf(" AND snr_db <= $%d", argIndex)
		args = append(args, *filter.MaxSnrDb)
		argIndex++
	}
	if filter.MinPrbUtilizationPct != nil {
		query += fmt.Sprintf(" AND prb_utilization_pct >= $%d", argIndex)
		args = append(args, *filter.MinPrbUtilizationPct)
		argIndex++
	}
	if filter.MaxPrbUtilizationPct != nil {
		query += fmt.Sprintf(" AND prb_utilization_pct <= $%d", argIndex)
		args = append(args, *filter.MaxPrbUtilizationPct)
		argIndex++
	}

	// Ordenação
	if len(filter.OrderBy) > 0 {
		query += " ORDER BY "
		for i, order := range filter.OrderBy {
			if i > 0 {
				query += ", "
			}
			query += order
		}
	} else {
		query += " ORDER BY measured_at DESC"
	}

	// Limitação e paginação
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++
	}
	if filter.Offset >= 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filter.Offset)
		argIndex++
	}

	// Executar a query
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("falha ao executar query de RadioKPI: %w", err)
	}
	defer rows.Close()

	// Processar resultados
	var kpis []*domain.RadioKPI
	for rows.Next() {
		kpi := &domain.RadioKPI{}
		err := rows.Scan(
			&kpi.ID, &kpi.TowerID, &kpi.SectorID, &kpi.CellTechnique,
			&kpi.MeasuredAt, &kpi.ReceivedAt,
			&kpi.TxPowerWatt, &kpi.TxPowerDbm, &kpi.RxPowerDbm, &kpi.SnrDb, &kpi.SinrDb, &kpi.RsrpDbm, &kpi.RsrqDbm,
			&kpi.ConnectedUEs, &kpi.MaxSupportedUEs, &kpi.PRBUtilizationPct, &kpi.ChannelOccupancyPct,
			&kpi.Ber, &kpi.Bler, &kpi.Fer, &kpi.CodecDropPct,
			&kpi.HoAttempt, &kpi.HoSuccess, &kpi.HoFail, &kpi.HoPingPong,
			&kpi.CallDropPct, &kpi.CallBlockPct, &kpi.PDCPSDULossPct, &kpi.RLCRetransPct,
			&kpi.SourceSystem, &kpi.CollectionIntervalSec, &kpi.RawData,
			&kpi.CreatedAt, &kpi.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("falha ao escanear RadioKPI: %w", err)
		}
		kpis = append(kpis, kpi)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("erro ao iterar resultados de RadioKPI: %w", err)
	}

	// Contar total sem limites para paginação informativa
	countQuery := "SELECT COUNT(*) FROM radio_kpi_staging WHERE 1=1"
	countArgs := []interface{}{}
	countArgIndex := 1

	// Aplicar os mesmos filtros de contagem (sem ordenação, limite, offset)
	if filter.TowerID != uuid.Nil {
		countQuery += fmt.Sprintf(" AND tower_id = $%d", countArgIndex)
		countArgs = append(countArgs, filter.TowerID)
		countArgIndex++
	}
	if filter.SectorID != nil {
		countQuery += fmt.Sprintf(" AND sector_id = $%d", countArgIndex)
		countArgs = append(countArgs, *filter.SectorID)
		countArgIndex++
	}
	if filter.CellTechnique != nil {
		countQuery += fmt.Sprintf(" AND cell_technique = $%d", countArgIndex)
		countArgs = append(countArgs, *filter.CellTechnique)
		countArgIndex++
	}
	if filter.MeasuredAtAfter != nil {
		countQuery += fmt.Sprintf(" AND measured_at >= $%d", countArgIndex)
		countArgs = append(countArgs, *filter.MeasuredAtAfter)
		countArgIndex++
	}
	if filter.MeasuredAtBefore != nil {
		countQuery += fmt.Sprintf(" AND measured_at <= $%d", countArgIndex)
		countArgs = append(countArgs, *filter.MeasuredAtBefore)
		countArgIndex++
	}
	if filter.ReceivedAtAfter != nil {
		countQuery += fmt.Sprintf(" AND received_at >= $%d", countArgIndex)
		countArgs = append(countArgs, *filter.ReceivedAtAfter)
		countArgIndex++
	}
	if filter.ReceivedAtBefore != nil {
		countQuery += fmt.Sprintf(" AND received_at <= $%d", countArgIndex)
		countArgs = append(countArgs, *filter.ReceivedAtBefore)
		countArgIndex++
	}
	if filter.MinRxPowerDbm != nil {
		countQuery += fmt.Sprintf(" AND rx_power_dbm >= $%d", countArgIndex)
		countArgs = append(countArgs, *filter.MinRxPowerDbm)
		countArgIndex++
	}
	if filter.MaxRxPowerDbm != nil {
		countQuery += fmt.Sprintf(" AND rx_power_dbm <= $%d", countArgIndex)
		countArgs = append(countArgs, *filter.MaxRxPowerDbm)
		countArgIndex++
	}
	if filter.MinSnrDb != nil {
		countQuery += fmt.Sprintf(" AND snr_db >= $%d", countArgIndex)
		countArgs = append(countArgs, *filter.MinSnrDb)
		countArgIndex++
	}
	if filter.MaxSnrDb != nil {
		countQuery += fmt.Sprintf(" AND snr_db <= $%d", countArgIndex)
		countArgs = append(countArgs, *filter.MaxSnrDb)
		countArgIndex++
	}
	if filter.MinPrbUtilizationPct != nil {
		countQuery += fmt.Sprintf(" AND prb_utilization_pct >= $%d", countArgIndex)
		countArgs = append(countArgs, *filter.MinPrbUtilizationPct)
		countArgIndex++
	}
	if filter.MaxPrbUtilizationPct != nil {
		countQuery += fmt.Sprintf(" AND prb_utilization_pct <= $%d", countArgIndex)
		countArgs = append(countArgs, *filter.MaxPrbUtilizationPct)
		countArgIndex++
	}

	var total int
	err = r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("falha ao contar RadioKPIs: %w", err)
	}

	return kpis, total, nil
}

// DeleteOlderThan remove KPIs mais antigos que um determinado tempo (para retenção de dados).
func (r *radioKPIRepository) DeleteOlderThan(ctx context.Context, olderThan time.Time) error {
	if olderThan.IsZero() {
		return errors.New("olderThan timestamp não pode ser zero")
	}

	query := `DELETE FROM radio_kpi_staging WHERE received_at < $1`
	result, err := r.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return fmt.Errorf("falha ao excluir RadioKPIs antigos: %w", err)
	}

	_, err = result.RowsAffected()
	if err != nil {
		return fmt.Errorf("falha ao obter linhas afetadas: %w", err)
	}

	// Log informativo (em produção, usar logger apropriado)
	// log.Infof("Removidos %d RadioKPIs antigos (antes de %v)", rowsAffected, olderThan)

	return nil
}

// GetByTowerAndSector retorna o KPI mais recente para uma torre/setor/técnica específica.
func (r *radioKPIRepository) GetByTowerAndSector(ctx context.Context, towerID uuid.UUID, sectorID string, technique string) (*domain.RadioKPI, error) {
	query := `
		SELECT
			id, tower_id, sector_id, cell_technique,
			measured_at, received_at,
			tx_power_watt, tx_power_dbm, rx_power_dbm, snr_db, sinr_db, rsrp_dbm, rsrq_db,
			connected_ues, max_supported_ues, prb_utilization_pct, channel_occupancy_pct,
			ber, bler, fer, codec_drop_pct,
			ho_attempt, ho_success, ho_fail, ho_ping_pong,
			call_drop_pct, call_block_pct, pdcp_sdu_loss_pct, rlc_retrans_pct,
			source_system, collection_interval_sec, raw_data,
			created_at, updated_at
		FROM radio_kpi_staging
		WHERE tower_id = $1 AND sector_id = $2 AND cell_technique = $3
		ORDER BY measured_at DESC
		LIMIT 1
	`

	kpi := &domain.RadioKPI{}
	err := r.db.QueryRowContext(ctx, query, towerID, sectorID, technique).Scan(
		&kpi.ID, &kpi.TowerID, &kpi.SectorID, &kpi.CellTechnique,
		&kpi.MeasuredAt, &kpi.ReceivedAt,
		&kpi.TxPowerWatt, &kpi.TxPowerDbm, &kpi.RxPowerDbm, &kpi.SnrDb, &kpi.SinrDb, &kpi.RsrpDbm, &kpi.RsrqDbm,
		&kpi.ConnectedUEs, &kpi.MaxSupportedUEs, &kpi.PRBUtilizationPct, &kpi.ChannelOccupancyPct,
		&kpi.Ber, &kpi.Bler, &kpi.Fer, &kpi.CodecDropPct,
		&kpi.HoAttempt, &kpi.HoSuccess, &kpi.HoFail, &kpi.HoPingPong,
		&kpi.CallDropPct, &kpi.CallBlockPct, &kpi.PDCPSDULossPct, &kpi.RLCRetransPct,
		&kpi.SourceSystem, &kpi.CollectionIntervalSec, &kpi.RawData,
		&kpi.CreatedAt, &kpi.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Não encontrado - não é erro
		}
		return nil, fmt.Errorf("falha ao buscar RadioKPI por torre/setor: %w", err)
	}

	return kpi, nil
}
