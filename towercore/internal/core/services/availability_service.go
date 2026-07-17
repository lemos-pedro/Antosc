package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"towercore/internal/core/interfaces"
)

// AvailabilityService calcula a disponibilidade de uma torre num
// período, conforme a fórmula em definições.md:
//
//	Disponibilidade (%) = ((Tempo total - Downtime) / Tempo total) * 100
//
// Downtime é medido exclusivamente a partir de eventos type=failure
// (interrupção real) — eventos type=alarm (risco/degradação) não
// entram no cálculo, conforme a distinção Alarme vs Falha do domínio.
// A maioria dos sites reporta falhas via Nagios (NagiosIngestService),
// não via SNMP.
type AvailabilityService struct {
	events interfaces.EventRepository
	towers interfaces.TowerRepository
}

func NewAvailabilityService(events interfaces.EventRepository, towers interfaces.TowerRepository) *AvailabilityService {
	return &AvailabilityService{events: events, towers: towers}
}

// Calculate devolve a disponibilidade (%) da torre nos últimos
// windowDays. Se a torre foi criada há menos tempo que a janela pedida,
// o cálculo usa created_at como início — para não penalizar uma torre
// recém-onboarded com um denominador maior do que o tempo em que
// esteve de facto monitorizada.
func (s *AvailabilityService) Calculate(ctx context.Context, towerID string, windowDays int) (float64, error) {
	towerID = strings.TrimSpace(towerID)
	if towerID == "" {
		return 0, errors.New("tower_id is required")
	}
	if windowDays <= 0 {
		windowDays = 30
	}

	tower, err := s.towers.GetByID(ctx, towerID)
	if err != nil {
		return 0, err
	}

	windowEnd := time.Now().UTC()
	windowStart := windowEnd.AddDate(0, 0, -windowDays)
	if tower.CreatedAt.After(windowStart) {
		windowStart = tower.CreatedAt
	}

	totalSeconds := windowEnd.Sub(windowStart).Seconds()
	if totalSeconds <= 0 {
		// Torre criada agora mesmo: sem histórico ainda, assume 100%.
		return 100, nil
	}

	downtimeSeconds, err := s.events.SumFailureDowntime(ctx, towerID, windowStart, windowEnd)
	if err != nil {
		return 0, err
	}

	// Clamp defensivo: sobreposição/arredondamento não deve produzir
	// downtime > totalSeconds nem disponibilidade negativa.
	if downtimeSeconds > totalSeconds {
		downtimeSeconds = totalSeconds
	}
	if downtimeSeconds < 0 {
		downtimeSeconds = 0
	}

	availability := ((totalSeconds - downtimeSeconds) / totalSeconds) * 100
	return availability, nil
}