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

// RadioKPIService gerencia a ingestão e consulta de KPIs de rádio
// federados de NMS/EMS externos.
type RadioKPIService struct {
	repo interfaces.RadioKPIGenericRepository
}

// NewRadioKPIService cria um novo serviço de KPIs de rádio.
func NewRadioKPIService(repo interfaces.RadioKPIGenericRepository) *RadioKPIService {
	return &RadioKPIService{repo: repo}
}

// IngestKPIs aceita KPIs de rádio já processados de um sistema externo (NMS/EMS).
// Esta é a porta de entrada para federar dados de qualidade de serviço de rádio.
//
// Os dados devem já estar agregados/processados pelo sistema fonte.
//
// Exemplo de uso:
//   service.IngestKPIs(ctx, []*domain.RadioKPI{
//     {
 //       TowerID:     siteUUID,
//       SectorID:    "A",
//       CellTechnique: "LTE",
//       MeasuredAt:  time.Now(),
//       TxPowerDbm:    43.0,
//       RxPowerDbm:    -85.0,
//       SnrDb:       15.0,
//       ConnectedUEs:  42,
//       PRBUtilizationPct: 65.0,
//       ...
//     },
//   })
func (s *RadioKPIService) IngestKPIs(ctx context.Context, kpis []*domain.RadioKPI) error {
	if len(kpis) == 0 {
		return nil
	}

	// Definir timestamps se não fornecidos
	now := time.Now().UTC()
	for _, kpi := range kpis {
		if kpi.ID == uuid.Nil {
			kpi.ID = uuid.New()
		}
		if kpi.MeasuredAt.IsZero() {
			kpi.MeasuredAt = now
		}
		if kpi.ReceivedAt.IsZero() {
			kpi.ReceivedAt = now
		}

		// Validação básica
		if kpi.TowerID == uuid.Nil {
			return errors.New("tower_id é obrigatório em RadioKPI")
		}
		if kpi.CellTechnique == "" {
			return errors.New("cell_technique é obrigatório em RadioKPI")
		}
		if kpi.MeasuredAt.After(now.Add(5 * time.Minute)) {
			return errors.New("measured_at não pode estar no futuro")
		}
	}

	// Ingestar no repositório
	if err := s.repo.CreateMany(ctx, kpis); err != nil {
		return fmt.Errorf("falha ao ingestar KPIs de rádio: %w", err)
	}

	return nil
}

// GetLatestKPIs retorna o KPI mais recente por torre/setor/tecnologia.
// Útil para dashboards e consultas em tempo real.
func (s *RadioKPIService) GetLatestKPIs(ctx context.Context, towerID uuid.UUID, sectorID *string, technique *string) ([]*domain.RadioKPI, error) {
	filter := &domain.RadioKPIFilter{
		TowerID:     towerID,
		SectorID:    sectorID,
		CellTechnique: technique,
		Limit:         1, // apenas o mais recente por combinação torre/setor/técnica
		Offset:        0,
		OrderBy:       []string{"measured_at DESC"},
	}

	kpis, _, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("falha ao buscar KPIs de rádio: %w", err)
	}
	return kpis, nil
}

// GetKPIsHistorico retorna KPIs históricos para análise de tendências.
func (s *RadioKPIService) GetKPIsHistorico(ctx context.Context, filter *domain.RadioKPIFilter) ([]*domain.RadioKPI, int, error) {
	if filter == nil {
		filter = &domain.RadioKPIFilter{}
	}
	return s.repo.List(ctx, filter)
}
