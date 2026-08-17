// Package services contém implementações de lógica de negócio para varias entidades do domínio.
package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
)

// BackhaulInterfaceService gerencia a coleta e consulta de métricas de interfaces de backhaul.
type BackhaulInterfaceService struct {
	repo interfaces.BackhaulInterfaceRepository
}

// NewBackhaulInterfaceService cria um novo serviço de métricas de backhaul.
func NewBackhaulInterfaceService(repo interfaces.BackhaulInterfaceRepository) *BackhaulInterfaceService {
	return &BackhaulInterfaceService{repo: repo}
}

// CollectInterface aceita uma medição coletada de uma interface de backhaul.
// Esta é a porta de entrada para dados vindos de polling SNMP, scripts locais, ou agentes.
func (s *BackhaulInterfaceService) CollectInterface(ctx context.Context, iface *domain.BackhaulInterface) error {
	if iface == nil {
		return errors.New("interface cannot be nil")
	}

	// Validar campos obrigatórios
	if iface.TowerID == uuid.Nil {
		return errors.New("tower_id é obrigatório")
	}
	if iface.Name == "" {
		return errors.New("interface_name é obrigatório")
	}
	if iface.MeasuredAt.IsZero() {
		return errors.New("measured_at é obrigatório")
	}
	if iface.AdminStatus == "" {
		return errors.New("admin_status é obrigatório")
	}
	if iface.OperStatus == "" {
		return errors.New("oper_status é obrigatório")
	}
	if iface.MeasuredAt.After(time.Now().Add(5 * time.Minute)) {
		return errors.New("measured_at não pode estar no futuro")
	}

	// Salvar no repositório
	if err := s.repo.Create(ctx, iface); err != nil {
		return errors.Errorf("falha ao coletar medição de backhaul: %w", err)
	}

	return nil
}

// CollectMultipleInterfaces aceita múltiplas medições de uma vez (ótimo para batch de polling).
func (s *BackhaulInterfaceService) CollectMultipleInterfaces(ctx context.Context, ifaces []*domain.BackhaulInterface) error {
	if len(ifaces) == 0 {
		return nil
	}

	// Validar cada interface
	now := time.Now().UTC()
	for _, iface := range ifaces {
		if iface == nil {
			continue // Pular nils silenciosamente
		}
		if iface.TowerID == uuid.Nil {
			return errors.New("tower_id é obrigatório em uma das interfaces")
		}
		if iface.Name == "" {
			return errors.New("interface_name é obrigatório em uma das interfaces")
		}
		if iface.MeasuredAt.IsZero() {
			iface.MeasuredAt = now // Definir timestamp se não fornecido
		}
		if iface.AdminStatus == "" {
			return errors.New("admin_status é obrigatório em uma das interfaces")
		}
		if iface.OperStatus == "" {
			return errors.New("oper_status é obrigatório em uma das interfaces")
		}
		if iface.MeasuredAt.After(now.Add(5 * time.Minute)) {
			return errors.New("measured_at não pode estar no futuro em uma das interfaces")
		}
	}

	// Salvar todas de uma vez
	if err := s.repo.CreateMany(ctx, ifaces); err != nil {
		return errors.Errorf("falha ao coletar medições múltiplas de backhaul: %w", err)
	}

	return nil
}

// GetLatestInterface retorna a medição mais recente para uma torre/interface.
func (s *BackhaulInterfaceService) GetLatestInterface(ctx context.Context, towerID uuid.UUID, interfaceName string) (*domain.BackhaulInterface, error) {
	iface, err := s.repo.GetLatest(ctx, towerID, interfaceName)
	if err != nil {
		return nil, errors.Errorf("falha ao buscar medição mais recente de backhaul: %w", err)
	}
	return iface, nil
}

// GetInterfaceHistory retorna o histórico de medições para uma torre/interface.
func (s *BackhaulInterfaceService) GetInterfaceHistory(ctx context.Context, towerID uuid.UUID, interfaceName string, limit int, offset int, measuredAfter *time.Time, measuredBefore *time.Time) ([]*domain.BackhaulInterface, int, error) {
	filter := &domain.BackhaulInterfaceFilter{
		TowerID:         towerID,
		InterfaceName:   &interfaceName,
		Limit:           limit,
		Offset:          offset,
		OrderBy:         []string{"measured_at DESC"},
		MeasuredAtAfter: measuredAfter,
		MeasuredAtBefore: measuredBefore,
	}

	ifaces, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, 0, errors.Errorf("falha ao buscar histórico de backhaul: %w", err)
	}
	return ifaces, total, nil
}

// GetTowerBackhaulStatus retorna o status consolidado de backhaul para uma torre.
// Útil para dashboards e alertas - retorna o estado mais recente conhecido.
func (s *BackhaulInterfaceService) GetTowerBackhaulStatus(ctx context.Context, towerID uuid.UUID) ([]*domain.BackhaulInterface, error) {
	// Buscar todas as interfaces mais recentes para esta torre
	// Uma abordagem é buscar o histórico e agrupar por interface, pegando o mais recente de cada uma
	query := `
		SELECT DISTINCT ON (interface_name)
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
			tx_power_dbm, rx_power_dbm, optic_temp_c, optic_bias_current_ma,
			los_events, lof_events, lom_events,
			optic_wavelength_nm, optic_vendor, optic_part_number, optic_serial_number, optic_date_code,
			source_poller, collection_interval_sec, raw_data,
			created_at, updated_at
		FROM backhaul_interface_history
		WHERE tower_id = $1
		ORDER BY interface_name, measured_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, towerID)
	if err != nil {
		return nil, errors.Errorf("falha ao buscar status de backhaul da torre: %w", err)
	}
	defer rows.Close()

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
			&iface.TxPowerDbm, &iface.RxPowerDbm, &iface.OpticTempC, &iface.OpticBiasCurrentMa,
			&iface.LosEvents, &iface.LofEvents, &iface.LomEvents,
			&iface.OpticWavelengthNm, &iface.OpticVendor, &iface.OpticPartNumber, &iface.OpticSerialNumber, &iface.OpticDateCode,
			&iface.SourcePoller, &iface.CollectionIntervalSec, &iface.RawData,
			&iface.CreatedAt, &iface.UpdatedAt,
		)
		if err != nil {
			return nil, errors.Errorf("falha ao escanear interface de backhaul: %w", err)
		}
		ifaces = append(ifaces, iface)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Errorf("erro ao iterar interfaces de backhaul: %w", err)
	}

	return ifaces, nil
}