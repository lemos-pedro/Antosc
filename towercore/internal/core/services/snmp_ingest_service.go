package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"towercore/internal/adapters/snmp"
	"towercore/internal/core/domain"
)

type SNMPIngestService struct {
	metricService *MetricService
	eventService  *EventService
	profiles      map[string]snmp.Profile
	towerUpdater  TowerStatusUpdater
	ticketService *TicketService
}

type SNMPSnapshot struct {
	TowerID     string
	Vendor      string
	CollectedAt time.Time
	Samples     map[string]float64
}

func NewSNMPIngestService(
	metricService *MetricService,
	eventService *EventService,
	profiles map[string]snmp.Profile,
	towerUpdater TowerStatusUpdater,
	ticketService *TicketService,
) *SNMPIngestService {
	return &SNMPIngestService{
		metricService: metricService,
		eventService:  eventService,
		profiles:      profiles,
		towerUpdater:  towerUpdater,
		ticketService: ticketService,
	}
}

func (s *SNMPIngestService) Ingest(ctx context.Context, snap SNMPSnapshot) error {
	if strings.TrimSpace(snap.TowerID) == "" {
		return errors.New("tower_id is required")
	}
	if len(snap.Samples) == 0 {
		return errors.New("samples is required")
	}

	vendor := strings.ToLower(strings.TrimSpace(snap.Vendor))
	profile, ok := s.profiles[vendor]
	if !ok {
		return fmt.Errorf("unsupported vendor: %s", snap.Vendor)
	}

	normalized := make(map[string]float64)
	for _, md := range profile.Metrics {
		raw, exists := snap.Samples[md.OID]
		if !exists {
			continue
		}
		scale := md.Scale
		if scale == 0 {
			scale = 1
		}
		normalized[md.Key] = raw * scale
	}
	if len(normalized) == 0 {
		return errors.New("no mapped OIDs found for selected vendor profile")
	}

	metric := &domain.Metric{
		TowerID:     snap.TowerID,
		CollectedAt: snap.CollectedAt,
		Values:      normalized,
	}
	if err := s.metricService.Create(ctx, metric); err != nil {
		return err
	}

	// Primeiro passo: avaliar TODOS os alarmes deste ciclo e determinar o
	// estado agregado (hasCritical/hasWarning), sem ainda tocar em
	// eventos/tickets. Isto separa "o que a torre está a fazer agora"
	// (usado para status) de "o que precisamos de registar/notificar"
	// (eventos e tickets) — para que uma falha na segunda parte nunca
	// deixe o status da torre desatualizado.
	type triggeredAlarm struct {
		rule  snmp.AlarmRule
		value float64
	}
	var triggered []triggeredAlarm
	var toResolve []string // alarm_key cuja condição já não se verifica

	hasCritical := false
	hasWarning := false

	for _, ar := range profile.Alarms {
		v, ok := normalized[ar.Key]
		if !ok {
			continue
		}
		if ar.IgnoreZero && v == 0 {
			toResolve = append(toResolve, ar.Key) // garante que resolve se estava aberto
			continue
		}
		if !matchCondition(v, ar.Condition, ar.Threshold) {
			toResolve = append(toResolve, ar.Key)
			continue
		} else if ar.Severity == domain.EventSeverityWarning {
			hasWarning = true
		}
		triggered = append(triggered, triggeredAlarm{rule: ar, value: v})
	}

	// Segundo passo: status da torre é atualizado já aqui, ANTES de
	// eventos/tickets. Coleta teve sucesso -> a torre está viva; reflete
	// isso no status independentemente do que acontecer a seguir.
	newStatus := domain.TowerStatusOnline
	if hasCritical || hasWarning {
		newStatus = domain.TowerStatusDegraded
	}
	if s.towerUpdater != nil {
		if err := s.towerUpdater.UpdateStatus(ctx, snap.TowerID, newStatus); err != nil {
			return err
		}
	}

	// Terceiro passo: resolver alarmes que deixaram de se verificar.
	for _, key := range toResolve {
		if err := s.eventService.Resolve(ctx, snap.TowerID, key); err != nil {
			return err
		}
	}

	// Quarto passo: registar eventos novos/persistentes e abrir tickets
	// só na transição OK->alarme. Se isto falhar a meio (ex. erro no
	// ticket), o status já ficou correto no passo dois — não volta a
	// ficar "preso" em degraded sem alarme aberto.
	for _, ta := range triggered {
		event := &domain.Event{
			TowerID:    snap.TowerID,
			Type:       domain.EventTypeAlarm,
			Severity:   ta.rule.Severity,
			Message:    fmt.Sprintf("%s: %s=%.2f threshold=%.2f", ta.rule.Message, ta.rule.Key, ta.value, ta.rule.Threshold),
			OccurredAt: metric.CollectedAt,
		}

		createdEvent, isNew, err := s.eventService.CreateOrTouch(ctx, event, ta.rule.Key)
		if err != nil {
			return err
		}

		if isNew && s.ticketService != nil {
			if _, err := s.ticketService.Create(ctx, snap.TowerID, createdEvent.ID); err != nil {
				return err
			}
		}
	}

	return nil
}

func matchCondition(value float64, condition string, threshold float64) bool {
	switch strings.ToLower(condition) {
	case "gt":
		return value > threshold
	case "gte":
		return value >= threshold
	case "lt":
		return value < threshold
	case "lte":
		return value <= threshold
	case "eq":
		return value == threshold
	case "ne":
		return value != threshold
	default:
		return false
	}
}