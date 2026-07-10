package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
)

type EventService struct {
	repo interfaces.EventRepository
}

func NewEventService(repo interfaces.EventRepository) *EventService {
	return &EventService{repo: repo}
}

// Create mantém o comportamento original (usado por outras origens de
// eventos, ex. Nagios, que podem não ter uma alarm_key de deduplicação).
func (s *EventService) Create(ctx context.Context, event *domain.Event) error {
	if err := validateEvent(event); err != nil {
		return err
	}

	if event.ID == "" {
		id, err := newUUIDv4()
		if err != nil {
			return err
		}
		event.ID = id
	}

	now := time.Now().UTC()
	if event.OccurredAt.IsZero() {
		event.OccurredAt = now
	}
	event.CreatedAt = now
	if event.Status == "" {
		event.Status = domain.EventStatusOpen // ASSUNÇÃO: domain.EventStatusOpen = "open"
	}
	event.LastSeenAt = now

	return s.repo.Create(ctx, event)
}

// CreateOrTouch é o ponto de entrada para eventos gerados por avaliação de
// threshold (SNMPIngestService). Evita duplicar eventos para uma condição
// de alarme já ativa: se já existe um evento "open" para tower_id+alarmKey,
// só atualiza last_seen_at; caso contrário cria um evento novo.
//
// Devolve (evento, criouNovo, erro) — "criouNovo" é o sinal que o
// SNMPIngestService usa para decidir se deve abrir um Ticket.
func (s *EventService) CreateOrTouch(ctx context.Context, event *domain.Event, alarmKey string) (*domain.Event, bool, error) {
	if err := validateEvent(event); err != nil {
		return nil, false, err
	}
	alarmKey = strings.TrimSpace(alarmKey)
	if alarmKey == "" {
		return nil, false, errors.New("alarm_key is required for CreateOrTouch")
	}

	now := time.Now().UTC()

	// ASSUNÇÃO: interfaces.EventRepository precisa de um novo método
	// FindOpenByTowerAndAlarmKey. Ver interface abaixo.
	existing, err := s.repo.FindOpenByTowerAndAlarmKey(ctx, event.TowerID, alarmKey)
	if err != nil && !errors.Is(err, interfaces.ErrEventNotFound) {
		return nil, false, err
	}

	if existing != nil {
		existing.LastSeenAt = now
		if err := s.repo.TouchLastSeen(ctx, existing.ID, now); err != nil {
			return nil, false, err
		}
		existing.LastSeenAt = now
		return existing, false, nil
	}

	if event.ID == "" {
		id, genErr := newUUIDv4()
		if genErr != nil {
			return nil, false, genErr
		}
		event.ID = id
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = now
	}
	event.CreatedAt = now
	event.LastSeenAt = now
	event.Status = domain.EventStatusOpen
	event.AlarmKey = alarmKey

	if err := s.repo.Create(ctx, event); err != nil {
		return nil, false, err
	}
	return event, true, nil
}

// Resolve fecha um evento "open" quando a condição deixa de se verificar.
// Chamado pelo SNMPIngestService quando um alarme antes ativo já não
// corresponde à condição no ciclo atual.
func (s *EventService) Resolve(ctx context.Context, towerID, alarmKey string) error {
	now := time.Now().UTC()
	return s.repo.ResolveOpenByTowerAndAlarmKey(ctx, towerID, alarmKey, now)
}

func (s *EventService) List(ctx context.Context, filter interfaces.EventFilter) ([]domain.Event, int, error) {
	if filter.Limit <= 0 || filter.Limit > 200 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	return s.repo.List(ctx, filter)
}

func validateEvent(event *domain.Event) error {
	if strings.TrimSpace(event.TowerID) == "" {
		return errors.New("tower_id is required")
	}
	if strings.TrimSpace(event.Message) == "" {
		return errors.New("message is required")
	}
	switch event.Type {
	case domain.EventTypeFailure, domain.EventTypeAlarm, domain.EventTypeInfo:
	default:
		return errors.New("invalid type")
	}
	switch event.Severity {
	case domain.EventSeverityInfo, domain.EventSeverityWarning, domain.EventSeverityCritical:
	default:
		return errors.New("invalid severity")
	}
	return nil
}