package services

import (
	"context"
	"errors"
	"fmt"
	"log"
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

// collectionStatusUpdater — métodos opcionais no TowerService.
type collectionStatusUpdater interface {
	MarkCollectionSuccess(ctx context.Context, towerID string, collectedAt time.Time) error
}

// collectionFailureUpdater — opcional; se não existir, só se actualiza status offline.
type collectionFailureUpdater interface {
	MarkCollectionFailed(ctx context.Context, towerID string, errMsg string) error
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
	if snap.CollectedAt.IsZero() {
		snap.CollectedAt = time.Now().UTC()
	}

	vendor := strings.ToLower(strings.TrimSpace(snap.Vendor))
	profile, ok := s.profiles[vendor]
	if !ok {
		return fmt.Errorf("unsupported vendor: %s", snap.Vendor)
	}

	normalized := normalizeSamples(profile, snap.Samples)
	if len(normalized) == 0 {
		// Collect SNMP pode ter respondido, mas nenhum OID do perfil mapeou.
		// Isto NÃO é "torre morta" — é config/perfil. Não chamar MarkUnreachable.
		return errors.New("no mapped OIDs found for selected vendor profile")
	}

	metric := &domain.Metric{
		TowerID:     snap.TowerID,
		CollectedAt: snap.CollectedAt,
		Values:      normalized,
	}
	if err := s.metricService.Create(ctx, metric); err != nil {
		return fmt.Errorf("metric create: %w", err)
	}

	// --- Avaliar alarmes (agregado para status) ---
	type triggeredAlarm struct {
		rule  snmp.AlarmRule
		value float64
	}
	var triggered []triggeredAlarm
	var toResolve []string
	hasCritical, hasWarning := false, false

	for _, ar := range profile.Alarms {
		v, ok := normalized[ar.Key]
		if !ok {
			continue
		}
		if ar.IgnoreZero && v == 0 {
			toResolve = append(toResolve, ar.Key)
			continue
		}
		if !matchCondition(v, ar.Condition, ar.Threshold) {
			toResolve = append(toResolve, ar.Key)
			continue
		}
		switch ar.Severity {
		case domain.EventSeverityCritical:
			hasCritical = true
		case domain.EventSeverityWarning:
			hasWarning = true
		}
		triggered = append(triggered, triggeredAlarm{rule: ar, value: v})
	}

	// --- Status + collection SUCCESS (antes de eventos/tickets) ---
	newStatus := domain.TowerStatusOnline
	if hasCritical || hasWarning {
		newStatus = domain.TowerStatusDegraded
	}
	if s.towerUpdater != nil {
		if err := s.towerUpdater.UpdateStatus(ctx, snap.TowerID, newStatus); err != nil {
			return fmt.Errorf("update status: %w", err)
		}
		if u, ok := s.towerUpdater.(collectionStatusUpdater); ok {
			if err := u.MarkCollectionSuccess(ctx, snap.TowerID, metric.CollectedAt); err != nil {
				return fmt.Errorf("mark collection success: %w", err)
			}
		}
	}

	// --- Resolver alarmes que já não disparam (erros não abortam o resto) ---
	for _, key := range toResolve {
		if err := s.eventService.Resolve(ctx, snap.TowerID, key); err != nil {
			log.Printf("[snmp-ingest] resolve alarm_key=%s tower=%s err=%v", key, snap.TowerID, err)
		}
	}

	// --- CreateOrTouch + ticket só em transição nova ---
	for _, ta := range triggered {
		event := &domain.Event{
			TowerID:    snap.TowerID,
			Type:       domain.EventTypeAlarm,
			Severity:   ta.rule.Severity,
			Message:    fmt.Sprintf("%s: %s=%.2f threshold=%.2f", ta.rule.Message, ta.rule.Key, ta.value, ta.rule.Threshold),
			DataSource: "direct_snmp",
			OccurredAt: metric.CollectedAt,
		}
		createdEvent, isNew, err := s.eventService.CreateOrTouch(ctx, event, ta.rule.Key)
		if err != nil {
			log.Printf("[snmp-ingest] CreateOrTouch tower=%s key=%s err=%v", snap.TowerID, ta.rule.Key, err)
			continue
		}
		if isNew && s.ticketService != nil {
			if _, err := s.ticketService.Create(ctx, snap.TowerID, createdEvent.ID); err != nil {
				log.Printf("[snmp-ingest] ticket tower=%s key=%s err=%v", snap.TowerID, ta.rule.Key, err)
			}
		}
	}

	return nil
}

// MarkUnreachable — Collect SNMP falhou (timeout, auth, rede).
func (s *SNMPIngestService) MarkUnreachable(ctx context.Context, towerID string) error {
	return s.MarkUnreachableWithError(ctx, towerID, "snmp collect failed")
}

// MarkUnreachableWithError põe offline + collection_failed + last_collection_error.
func (s *SNMPIngestService) MarkUnreachableWithError(ctx context.Context, towerID, errMsg string) error {
	if strings.TrimSpace(towerID) == "" {
		return errors.New("tower_id is required")
	}
	if s.towerUpdater == nil {
		return nil
	}
	if u, ok := s.towerUpdater.(collectionFailureUpdater); ok {
		if err := u.MarkCollectionFailed(ctx, towerID, errMsg); err != nil {
			log.Printf("[snmp-ingest] MarkCollectionFailed tower=%s err=%v", towerID, err)
		}
	}
	return s.towerUpdater.UpdateStatus(ctx, towerID, domain.TowerStatusOffline)
}

// MarkIngestConfigError — SNMP respondeu, mas o perfil/OIDs falharam no Ingest.
// Grava o erro em collection_failed SEM marcar offline (não é falha de rede).
func (s *SNMPIngestService) MarkIngestConfigError(ctx context.Context, towerID, errMsg string) error {
	if strings.TrimSpace(towerID) == "" {
		return errors.New("tower_id is required")
	}
	if s.towerUpdater == nil {
		return nil
	}
	if u, ok := s.towerUpdater.(collectionFailureUpdater); ok {
		msg := strings.TrimSpace(errMsg)
		if msg != "" && !strings.HasPrefix(msg, "ingest:") {
			msg = "ingest: " + msg
		}
		return u.MarkCollectionFailed(ctx, towerID, msg)
	}
	return nil
}

func normalizeSamples(profile snmp.Profile, samples map[string]float64) map[string]float64 {
	normalized := make(map[string]float64)
	for _, md := range profile.Metrics {
		raw, exists := samples[md.OID]
		if !exists {
			continue
		}
		scale := md.Scale
		if scale == 0 {
			scale = 1
		}
		value := raw * scale
		if isIgnoredMetricValue(value, md.IgnoreValues) {
			continue
		}
		normalized[md.Key] = value
		if md.ZeroMeansNotTested && value == 0 {
			normalized[md.Key+"_not_tested"] = 1
		}
	}
	return normalized
}

func isIgnoredMetricValue(value float64, ignored []float64) bool {
	for _, candidate := range ignored {
		if value == candidate {
			return true
		}
	}
	return false
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