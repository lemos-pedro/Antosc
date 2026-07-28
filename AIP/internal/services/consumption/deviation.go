// Package consumption calcula o desvio entre o consumo real (ai_features) e
// as normas definidas pelo Controller (consumption_norms). Alimenta tanto o
// relatório do Financeiro como os alertas de não-conformidade.
package consumption

import (
	"context"
	"time"

	"github.com/antosc/aip/internal/repository/postgres"
)

// Deviation é o resultado da comparação entre consumo real e a norma, para
// um site/equipamento, num período. AverageActual é a média das leituras
// no período — assume-se que feature_name corresponde ao equipment_type da norma.
type Deviation struct {
	TowerID          string
	EquipmentType    string
	ExpectedValue    float64
	AverageActual    float64
	Unit             string
	TolerancePercent float64
	DeviationPercent float64
	WithinNorm       bool
	SampleCount      int
}

type Service interface {
	// ComputeForTower calcula o desvio de todas as normas ativas de uma torre, no período dado.
	ComputeForTower(ctx context.Context, towerID string, from, to time.Time) ([]Deviation, error)

	// Compare calcula o desvio em dois períodos distintos e devolve a variação entre eles,
	// por equipamento -- base da funcionalidade de "comparar épocas diferentes".
	Compare(ctx context.Context, towerID string, periodAFrom, periodATo, periodBFrom, periodBTo time.Time) ([]Comparison, error)
}

// Comparison é o resultado de comparar o consumo médio real do mesmo
// equipamento em dois períodos diferentes (ex: mês passado vs este mês).
type Comparison struct {
	EquipmentType    string
	Unit             string
	PeriodAAverage   float64
	PeriodBAverage   float64
	ChangeAbsolute   float64
	ChangePercent    float64
}

type service struct {
	norms    postgres.ConsumptionNormRepository
	features postgres.FeatureRepository
}

func NewService(norms postgres.ConsumptionNormRepository, features postgres.FeatureRepository) Service {
	return &service{norms: norms, features: features}
}

func (s *service) ComputeForTower(ctx context.Context, towerID string, from, to time.Time) ([]Deviation, error) {
	norms, err := s.norms.ListByTower(ctx, towerID)
	if err != nil {
		return nil, err
	}

	out := make([]Deviation, 0, len(norms))
	for _, n := range norms {
		// Convenção: o feature_name gravado pela ingestão corresponde ao
		// equipment_type da norma (ex: "generator", "battery", "grid").
		// Se a tua ingestão usar nomes diferentes, ajusta aqui o mapeamento.
		readings, err := s.features.ListByTowerAndRange(ctx, towerID, n.EquipmentType, from, to)
		if err != nil {
			return nil, err
		}
		if len(readings) == 0 {
			continue
		}

		var sum float64
		for _, r := range readings {
			sum += r.Value
		}
		avg := sum / float64(len(readings))

		deviationPct := 0.0
		if n.ExpectedValue != 0 {
			deviationPct = ((avg - n.ExpectedValue) / n.ExpectedValue) * 100
		}

		withinNorm := deviationPct >= -n.TolerancePercent && deviationPct <= n.TolerancePercent

		out = append(out, Deviation{
			TowerID:          towerID,
			EquipmentType:    n.EquipmentType,
			ExpectedValue:    n.ExpectedValue,
			AverageActual:    avg,
			Unit:             n.Unit,
			TolerancePercent: n.TolerancePercent,
			DeviationPercent: deviationPct,
			WithinNorm:       withinNorm,
			SampleCount:      len(readings),
		})
	}

	return out, nil
}

func (s *service) Compare(ctx context.Context, towerID string, periodAFrom, periodATo, periodBFrom, periodBTo time.Time) ([]Comparison, error) {
	devsA, err := s.ComputeForTower(ctx, towerID, periodAFrom, periodATo)
	if err != nil {
		return nil, err
	}
	devsB, err := s.ComputeForTower(ctx, towerID, periodBFrom, periodBTo)
	if err != nil {
		return nil, err
	}

	byEquipment := make(map[string]Deviation, len(devsB))
	for _, d := range devsB {
		byEquipment[d.EquipmentType] = d
	}

	out := make([]Comparison, 0, len(devsA))
	for _, a := range devsA {
		b, ok := byEquipment[a.EquipmentType]
		if !ok {
			continue
		}

		changeAbs := b.AverageActual - a.AverageActual
		changePct := 0.0
		if a.AverageActual != 0 {
			changePct = (changeAbs / a.AverageActual) * 100
		}

		out = append(out, Comparison{
			EquipmentType:  a.EquipmentType,
			Unit:           a.Unit,
			PeriodAAverage: a.AverageActual,
			PeriodBAverage: b.AverageActual,
			ChangeAbsolute: changeAbs,
			ChangePercent:  changePct,
		})
	}

	return out, nil
}
