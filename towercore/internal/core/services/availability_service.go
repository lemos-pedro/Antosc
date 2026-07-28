package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"towercore/internal/core/interfaces"
)

// AvailabilityService calcula a disponibilidade de uma torre num
// período, conforme a fórmula em definições.md.
//
// IMPORTANTE: downtime não é medido só por eventos type=failure
// explícitos. Se o pipeline de coleta em si estiver interrompido
// (snmp_enabled=false, hostname Nagios inválido, torre nunca
// reportou dados), nenhum evento de falha chega a ser criado — e
// sem isso, a fórmula original assumia silenciosamente 100%, o que
// mascarava sites offline há semanas. Por isso cruzamos também com
// a última métrica recebida (heartbeat) como segundo sinal de
// downtime.
type AvailabilityService struct {
	events  interfaces.EventRepository
	towers  interfaces.TowerRepository
	metrics interfaces.MetricRepository
}

func NewAvailabilityService(events interfaces.EventRepository, towers interfaces.TowerRepository, metrics interfaces.MetricRepository) *AvailabilityService {
	return &AvailabilityService{events: events, towers: towers, metrics: metrics}
}

// staleThreshold: se a última métrica recebida for mais antiga que isto
// (ou nunca existiu), consideramos a torre sem sinal de vida — o pipeline
// de coleta está interrompido, não o serviço da torre necessariamente,
// mas do ponto de vista operacional isso é indistinguível de downtime.
const staleThreshold = 2 * time.Hour

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
		return 100, nil
	}

	downtimeSeconds, err := s.events.SumFailureDowntime(ctx, towerID, windowStart, windowEnd)
	if err != nil {
		return 0, err
	}

	// --- Downtime por ausência de heartbeat ---
	// Última métrica real recebida desta torre, independentemente de
	// eventos. Se nunca houve, ou a última é mais antiga que
	// staleThreshold, tratamos o tempo desde então (ou a janela toda,
	// se nunca houve dado) como downtime adicional.
	lastSeen, err := s.metrics.LastCollectedAt(ctx, towerID)
	if err != nil {
		return 0, err
	}

	var silenceDowntime float64
	switch {
	case lastSeen == nil:
		// Nunca recebemos nenhuma métrica desta torre: toda a janela
		// conta como downtime por falta de sinal.
		silenceDowntime = totalSeconds
	case lastSeen.Before(windowEnd.Add(-staleThreshold)):
		// Última métrica está mais antiga que o threshold: o tempo
		// desde a última métrica até agora conta como downtime,
		// recortado para dentro da janela.
		since := *lastSeen
		if since.Before(windowStart) {
			since = windowStart
		}
		silenceDowntime = windowEnd.Sub(since).Seconds()
	}

	// Usa o maior dos dois — não soma, para não duplicar o mesmo
	// intervalo caso um evento failure já cubra o mesmo período de
	// silêncio (ex: SNMPIngestService que ainda consegue detetar e
	// registar a falha corretamente).
	if silenceDowntime > downtimeSeconds {
		downtimeSeconds = silenceDowntime
	}

	if downtimeSeconds > totalSeconds {
		downtimeSeconds = totalSeconds
	}
	if downtimeSeconds < 0 {
		downtimeSeconds = 0
	}

	availability := ((totalSeconds - downtimeSeconds) / totalSeconds) * 100
	return availability, nil
}