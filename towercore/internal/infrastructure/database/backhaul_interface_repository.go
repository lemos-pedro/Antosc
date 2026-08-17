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

// backhaulInterfaceRepository implementa a interface BackhaulInterfaceRepository usando PostgreSQL.
type backhaulInterfaceRepository struct {
	db *sql.DB
}

// NewBackhaulInterfaceRepository cria um novo repositório de métricas de backhaul.
func NewBackhaulInterfaceRepository(db *sql.DB) domain.BackhaulInterfaceRepository {
	return &backhaulInterfaceRepository{db: db}
}

// Create insere uma nova medição de interface de backhaul.
func (r *backhaulInterfaceRepository) Create(ctx context.Context, iface *domain.BackhaulInterface) error {
	if iface == nil {
		return errors.New("interface cannot be nil")
	}

	// Gerar ID se não fornecido
	if iface.ID == uuid.Nil {
		iface.ID = uuid.New()
	}
	if iface.InterfaceID == "" {
		iface.InterfaceID = uuid.New().String()
	}

	// Definir timestamps se não fornecidos
	now := time.Now().UTC()
	if iface.MeasuredAt.IsZero() {
		iface.MeasuredAt = now
	}
	if iface.ReceivedAt.IsZero() {
		iface.ReceivedAt = now
	}

	query := `
		INSERT INTO backhaul_interface_history (
			id, tower_id, interface_id,
			interface_name, interface_description,
			measured_at, received_at,
			admin_status, oper_status, last_change,
			if_type, if_speed_bigint, if_speed_mbps, duplex_mode, media_type, connector_type,
			in_octets, out_octets,
			in_unicast_pkts, out_unicast_pkts,
			in_discards, out_discards,
			in_errors, out_errors,
			in_unknown_protos,
			in_frame_errors, out_frame_errors,
			in_jabbers, out_jabbers,
			in_fragments, out_fragments,
			utilization_pct,
			in_bandwidth_mbps, out_bandwidth_mbps,
			avg_latency_ms, min_latency_ms, max_latency_ms,
			loss_pct, jitter_ms,
			source_poller, collection_interval_sec, raw_data,
			created_at, updated_at
		) VALUES (
			$1, $2, $3,
			$4, $5,
			$6, $7,
			$8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17,
			$18, $19,
			$20, $21,
			$22, $23,
			$24, $25,
			$26, $27,
			$28, $29,
			$30, $31,
			$32, $33,
			$34, $35, $36, $37, $38,
			$39, $40,
			$41, $42,
			$43,
			$44, $45
		)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		iface.ID, iface.TowerID, iface.InterfaceID,
		iface.Name, iface.Description,
		iface.MeasuredAt, iface.ReceivedAt,
		iface.AdminStatus, iface.OperStatus, iface.LastChange,
		iface.IfType, nil, iface.IfSpeedMbps, iface.DuplexMode, iface.MediaType, iface.ConnectorType,
		iface.InOctets, iface.OutOctets,
		iface.InUnicastPkts, iface.OutUnicastPkts,
		iface.InDiscards, iface.OutDiscards,
		iface.InErrors, iface.OutErrors,
		iface.InUnknownProtos,
		iface.InFrameErrors, iface.OutFrameErrors,
		iface.InJabbers, iface.OutJabbers,
		iface.InFragments, iface.OutFragments,
		iface.UtilizationPct,
		iface.InBandwidthMbps, iface.OutBandwidthMbps,
		iface.AvgLatencyMs, iface.MinLatencyMs, iface.MaxLatencyMs,
		iface.LossPct, iface.JitterMs,
		iface.SourcePoller, iface.CollectionIntervalSec, iface.RawData,
		iface.CreatedAt, iface.UpdatedAt,
	)

	if err != nil {
		return errors.Errorf("falha ao inserir medição de backhaul: %w", err)
	}

	return nil
}

// CreateMany insere múltiplas medições de interface de backhaul de uma vez.
func (r *backhaulInterfaceRepository) CreateMany(ctx context.Context, ifaces []*domain.BackhaulInterface) error {
	if len(ifaces) == 0 {
		return nil
	}

	query := `
		INSERT INTO backhaul_interface_history (
			id, tower_id, interface_id,
			interface_name, interface_description,
			measured_at, received_at,
			admin_status, oper_status, last_change,
			if_type, if_speed_bigint, if_speed_mbps, duplex_mode, media_type, connector_type,
			in_octets, out_octets,
			in_unicast_pkts, out_unicast_pkts,
			in_discards, out_discards,
			in_errors, out_errors,
			in_unknown_protos,
			in_frame_errors, out_frame_errors,
			in_jabbers, out_jabbers,
			in_fragments, out_fragments,
			utilization_pct,
			in_bandwidth_mbps, out_bandwidth_mbps,
			avg_latency_ms, min_latency_ms, max_latency_ms,
			loss_pct, jitter_ms,
			source_poller, collection_interval_sec, raw_data,
			created_at, updated_at
		) VALUES (
			$1, $2, $3,
			$4, $5,
			$6, $7,
			$8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17,
			$18, $19,
			$20, $21,
			$22, $23,
			$24, $25,
			$26, $27,
			$28, $29,
			$30, $31,
			$32, $33,
			$34, $35, $36, $37, $38,
			$39, $40,
			$41, $42,
			$43,
			$44, $45
		)
	`

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Errorf("falha ao iniciar transação: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			if errTx := tx.Commit(); errTx != nil {
				err = errors.Errorf("falha ao commitar transação: %w", errTx)
			}
		}
	}()

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return errors.Errorf("falha ao preparar statement: %w", err)
	}
	defer stmt.Close()

	for _, iface := range ifaces {
		// Gerar ID se não fornecido
		if iface.ID == uuid.Nil {
			iface.ID = uuid.New()
		}
		if iface.InterfaceID == "" {
			iface.InterfaceID = uuid.New().String()
		}

		// Definir timestamps se não fornecidos
		now := time.Now().UTC()
		if iface.MeasuredAt.IsZero() {
			iface.MeasuredAt = now
			if iface.ReceivedAt.IsZero() {
				iface.ReceivedAt = now
			}
		}

		_, err = stmt.ExecContext(
			ctx,
			iface.ID, iface.TowerID, iface.InterfaceID,
			iface.Name, iface.Description,
			iface.MeasuredAt, iface.ReceivedAt,
			iface.AdminStatus, iface.OperStatus, iface.LastChange,
			iface.IfType, nil, iface.IfSpeedMbps, iface.DuplexMode, iface.MediaType, iface.ConnectorType,
			iface.InOctets, iface.OutOctets,
			iface.InUnicastPkts, iface.OutUnicastPkts,
			iface.InDiscards, iface.OutDiscards,
			iface.InErrors, iface.OutErrors,
			iface.InUnknownProtos,
			iface.InFrameErrors, iface.OutFrameErrors,
			iface.InJabbers, iface.OutJabbers,
			iface.InFragments, iface.OutFragments,
			iface.UtilizationPct,
			iface.InBandwidthMbps, iface.OutBandwidthMbps,
			iface.AvgLatencyMs, iface.MinLatencyMs, iface.MaxLatencyMs,
			iface.LossPct, iface.JitterMs,
			iface.SourcePoller, iface.CollectionIntervalSec, iface.RawData,
			iface.CreatedAt, iface.UpdatedAt,
		)

		if err != nil {
			return errors.Errorf("falha ao inserir medição de backhaul: %w", err)
		}
	}

	return nil
}

// List retorna medições de backhaul baseado em filtros.
func (r *backhaulInterfaceRepository) List(ctx context.Context, filter *domain.BackhaulInterfaceFilter) ([]*domain.BackhaulInterface, int, error) {
	if filter == nil {
		filter = &domain.BackhaulInterfaceFilter{}
	}

	query := `
		SELECT
			id, tower_id, interface_id,
			interface_name, interface_description,
			measured_at, received_at,
			admin_status, oper_status, last_change,
			if_type, if_speed_bigint, if_speed_mbps, duplex_mode, media_type, connector_type,
			in_octets, out_octets,
			in_unicast_pkts, out_unicast_pkts,
			in_discards, out_discards,
			in_errors, out_errors,
			in_unknown_protos,
			in_frame_errors, out_frame_errors,
			in_jabbers, out_jabbers,
			in_fragments, out_fragments,
			utilization_pct,
			in_bandwidth_mbps, out_bandwidth_mbps,
			avg_latency_ms, min_latency_ms, max_latency_ms,
			loss_pct, jitter_ms,
			source_poller, collection_interval_sec, raw_data,
			created_at, updated_at
		FROM backhaul_interface_history
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
	if filter.InterfaceName != nil {
		query += fmt.Sprintf(" AND interface_name = $%d", argIndex)
		args = append(args, *filter.InterfaceName)
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
	if filter.AdminStatus != nil {
		query += fmt.Sprintf(" AND admin_status = $%d", argIndex)
		args = append(args, *filter.AdminStatus)
		argIndex++
	}
	if filter.OperStatus != nil {
		query += fmt.Sprintf(" AND oper_status = $%d", argIndex)
		args = append(args, *filter.OperStatus)
		argIndex++
	}
	if filter.MinUtilizationPct != nil {
		query += fmt.Sprintf(" AND utilization_pct >= $%d", argIndex)
		args = append(args, *filter.MinUtilizationPct)
		argIndex++
	}
	if filter.MaxUtilizationPct != nil {
		query += fmt.Sprintf(" AND utilization_pct <= $%d", argIndex)
		args = append(args, *filter.MaxUtilizationPct)
		argIndex++
	}
	if filter.MinLatencyMs != nil {
		query += fmt.Sprintf(" AND avg_latency_ms >= $%d", argIndex)
		args = append(args, *filter.MinLatencyMs)
		argIndex++
	}
	if filter.MaxLatencyMs != nil {
		query += fmt.Sprintf(" AND avg_latency_ms <= $%d", argIndex)
		args = append(args, *filter.MaxLatencyMs)
		argIndex++
	}
	if filter.MaxLossPct != nil {
		query += fmt.Sprintf(" AND loss_pct <= $%d", argIndex)
		args = append(args, *filter.MaxLossPct)
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
		return nil, 0, errors.Errorf("falha ao executar query de backhaul: %w", err)
	}
	defer rows.Close()

	// Processar resultados
	var ifaces []*domain.BackhaulInterface
	for rows.Next() {
		iface := &domain.BackhaulInterface{}
		err := rows.Scan(
			&iface.ID, &iface.TowerID, &iface.InterfaceID,
			&iface.Name, &iface.Description,
			&iface.MeasuredAt, &iface.ReceivedAt,
			&iface.AdminStatus, &iface.OperStatus, &iface.LastChange,
			&iface.IfType, &iface.IfSpeedBigint, &iface.IfSpeedMbps, &iface.DuplexMode, &iface.MediaType, &iface.ConnectorType,
			&iface.InOctets, &iface.OutOctets,
			&iface.InUnicastPkts, &iface.OutUnicastPkts,
			&iface.InDiscards, &iface.OutDiscards,
			&iface.InErrors, &iface.OutErrors,
			&iface.InUnknownProtos,
			&iface.InFrameErrors, &iface.OutFrameErrors,
			&iface.InJabbers, &iface.OutJabbers,
			&iface.InFragments, &iface.OutFragments,
			&iface.UtilizationPct,
			&iface.InBandwidthMbps, &iface.OutBandwidthMbps,
			&iface.AvgLatencyMs, &iface.MinLatencyMs, &iface.MaxLatencyMs,
			&iface.LossPct, &iface.JitterMs,
			&iface.SourcePoller, &iface.CollectionIntervalSec, &iface.RawData,
			&iface.CreatedAt, &iface.UpdatedAt,
		)
		if err != nil {
			return nil, 0, errors.Errorf("falha ao escanear medição de backhaul: %w", err)
		}
		ifaces = append(ifaces, iface)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, errors.Errorf("erro ao iterar resultados de backhaul: %w", err)
	}

	// Contar total sem limites para paginação informativa
	countQuery := "SELECT COUNT(*) FROM backhaul_interface_history WHERE 1=1"
	countArgs := []interface{}
	countArgIndex := 1

	// Aplicar os mesmos filtros de contagem (sem ordenação, limite, offset)
	if filter.TowerID != uuid.Nil {
		countQuery += fmt.Sprintf(" AND tower_id = $%d", countArgIndex)
		countArgs = append(countArgs, filter.TowerID)
		countArgIndex++
	}
	if filter.InterfaceName != nil {
		countQuery += fmt.Sprintf(" AND interface_name = $%d", countArgIndex)
		args = append(args, *filter.InterfaceName)
		countArgIndex++
	}
	if filter.MeasuredAtAfter != nil {
		countQuery += fmt.Sprintf(" AND measured_at >= $%d", countArgIndex)
		countArgs = append(args, *filter.MeasuredAtAfter)
		countArgIndex++
	}
	if filter.MeasuredAtBefore != nil {
		countQuery += fmt.Sprintf(" AND measured_at <= $%d", countArgIndex)
		countArgs = append(args, *filter.MeasuredAtBefore)
		countArgIndex++
	}
	if filter.ReceivedAtAfter != nil {
		countQuery += fmt.Sprintf(" AND received_at >= $%d", countArgIndex)
		countArgs = append(args, *filter.ReceivedAtAfter)
		countArgIndex++
	}
	if filter.ReceivedAtBefore != nil {
		countQuery += fmt.Sprintf(" AND received_at <= $%d", countArgIndex)
		countArgs = append(args, *filter.ReceivedAtBefore)
		countArgIndex++
	}
	if filter.AdminStatus != nil {
		countQuery += fmt.Sprintf(" AND admin_status = $%d", countArgIndex)
		args = append(args, *filter.AdminStatus)
		countArgIndex++
	}
	if filter.OperStatus != nil {
		countQuery += fmt.Sprintf(" AND oper_status = $%d", countArgIndex)
		args = append(args, *filter.OperStatus)
		countArgIndex++
	}
	if filter.MinUtilizationPct != nil {
		countQuery += fmt.Sprintf(" AND utilization_pct >= $%d", countArgIndex)
		countArgs = append(args, *filter.MinUtilizationPct)
		countArgIndex++
	}
	if filter.MaxUtilizationPct != nil {
		countQuery += fmt.Sprintf(" AND utilization_pct <= $%d", countArgIndex)
		args = append(args, *filter.MaxUtilizationPct)
		countArgIndex++
	}
	if filter.MinLatencyMs != nil {
		countQuery += fmt.Sprintf(" AND avg_latency_ms >= $%d", countArgIndex)
		args = append(args, *filter.MinLatencyMs)
		countArgIndex++
	}
	if filter.MaxLatencyMs != nil {
		countQuery += fmt.Sprintf(" AND avg_latency_ms <= $%d", countArgIndex)
		args = append(args, *filter.MaxLatencyMs)
		countArgIndex++
	}
	if filter.MaxLossPct != nil {
		// Pular filtro de perda para contagem - vamos contar tudo
	}

	var total int
	err = r.db.QueryContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, errors.Errorf("falha ao contar medições de backhaul: %w", err)
	}

	return ifaces, total, nil
}

// GetLatest retorna a medição mais recente para uma torre/interface específica.
func (r *backhaulInterfaceRepository) GetLatest(ctx context.Context, towerID uuid.UUID, interfaceName string) (*domain.BackhaulInterface, error) {
	query := `
		SELECT
			id, tower_id, interface_id,
			interface_name, interface_description,
			measured_at, received_at,
			admin_status, oper_status, last_change,
			if_type, if_speed_bigint, if_speed_mbps, duplex_mode, media_type, connector_type,
			in_octets, out_octets,
			in_unicast_pkts, out_unicast_pkts,
			in_discards, out_discards,
			in_errors, out_errors,
			in_unknown_protos,
			in_frame_errors, out_frame_errors,
			in_jabbers, out_jabbers,
			in_fragments, out_fragments,
			utilization_pct,
			in_bandwidth_mbps, out_bandwidth_mbps,
			avg_latency_ms, min_latency_ms, max_latency_ms,
			loss_pct, jitter_ms,
			source_poller, collection_interval_sec, raw_data,
			created_at, updated_at
		FROM backhaul_interface_history
		WHERE tower_id = $1 AND interface_name = $2
		ORDER BY measured_at DESC
		LIMIT 1
	`

	iface := &domain.BackhaulInterface{}
	err := r.db.QueryContext(ctx, query, towerID, interfaceName).Scan(
		&iface.ID, &iface.TowerID, &iface.InterfaceID,
		&iface.Name, &iface.Description,
		&iface.MeasuredAt, &iface.ReceivedAt,
		&iface.AdminStatus, &iface.OperStatus, &iface.LastChange,
		&iface.IfType, &iface.IfSpeedBigint, &iface.IfSpeedMbps, &iface.DuplexMode, &iface.MediaType, &iface.ConnectorType,
		&iface.InOctets, &iface.OutOctets,
		&iface.InUnicastPkts, &iface.OutUnicastPkts,
		&iface.InDiscards, &iface.OutDiscards,
		&iface.InErrors, &iface.OutErrors,
		&iface.InUnknownProtos,
		&iface.InFrameErrors, &iface.OutFrameErrors,
		&iface.InJabbers, &iface.OutJabbers,
		&iface.InFragments, &iface.OutFragments,
		&iface.UtilizationPct,
		&iface.InBandwidthMbps, &iface.OutBandwidthMbps,
		&iface.AvgLatencyMs, &iface.MinLatencyMs, &iface.MaxLatencyMs,
		&iface.LossPct, &iface.JitterMs,
		&iface.SourcePoller, &iface.CollectionIntervalSec, &iface.RawData,
		&iface.CreatedAt, &iface.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Não encontrado - não é erro
		}
		return nil, errors.Errorf("falha ao buscar medição mais recente de backhaul: %w", err)
	}

	return iface, nil
}

// DeleteOlderThan remove medições mais antigas que um determinado tempo (para retenção de dados).
func (r *backhaulInterfaceRepository) DeleteOlderThan(ctx context.Context, olderThan time.Time) error {
	if olderThan.IsZero() {
		return errors.New("olderThan timestamp não pode ser zero")
	}

	query := `DELETE FROM backhaul_interface_history WHERE received_at < $1`
	result, err := r.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return errors.Errorf("falha ao excluir medições antigas de backhaul: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Errorf("falha ao obter linhas afetadas: %w", err)
	}

	return nil
}