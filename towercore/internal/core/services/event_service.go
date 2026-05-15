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

func (s *EventService) Create(ctx context.Context, event *domain.Event) error {
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

	return s.repo.Create(ctx, event)
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
