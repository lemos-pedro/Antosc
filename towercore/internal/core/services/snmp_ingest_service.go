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
}

type SNMPSnapshot struct {
	TowerID     string
	Vendor      string
	CollectedAt time.Time
	Samples     map[string]float64
}

func NewSNMPIngestService(metricService *MetricService, eventService *EventService, profiles map[string]snmp.Profile) *SNMPIngestService {
	return &SNMPIngestService{
		metricService: metricService,
		eventService:  eventService,
		profiles:      profiles,
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

	for _, ar := range profile.Alarms {
		v, ok := normalized[ar.Key]
		if !ok {
			continue
		}
		if !matchCondition(v, ar.Condition, ar.Threshold) {
			continue
		}

		event := &domain.Event{
			TowerID:    snap.TowerID,
			Type:       domain.EventTypeAlarm,
			Severity:   ar.Severity,
			Message:    fmt.Sprintf("%s: %s=%.2f threshold=%.2f", ar.Message, ar.Key, v, ar.Threshold),
			OccurredAt: metric.CollectedAt,
		}
		if err := s.eventService.Create(ctx, event); err != nil {
			return err
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
	default:
		return false
	}
}
