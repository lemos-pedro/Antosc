package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
)

// TowerStatusUpdater é o port mínimo que este service precisa para
// refletir o estado do Nagios na torre. Implementado por *TowerService
// via TowerService.UpdateStatus.
type TowerStatusUpdater interface {
	UpdateStatus(ctx context.Context, towerID string, status domain.TowerStatus) error
}

type NagiosIngestService struct {
	eventService *EventService
	towerUpdater TowerStatusUpdater
}

func NewNagiosIngestService(eventService *EventService, towerUpdater TowerStatusUpdater) *NagiosIngestService {
	return &NagiosIngestService{
		eventService: eventService,
		towerUpdater: towerUpdater,
	}
}

// Ingest aplica a regra de definições.md: down/unreachable é Falha real
// (interrupção), não Alarme (risco). "up" só atualiza o status, sem
// evento — não queremos ruído a cada poll quando está tudo normal.
func (s *NagiosIngestService) Ingest(ctx context.Context, towerID string, status interfaces.HostStatus) error {
	if strings.TrimSpace(towerID) == "" {
		return errors.New("tower_id is required")
	}

	domainStatus, eventNeeded, severity := mapNagiosState(status.State)

	if err := s.towerUpdater.UpdateStatus(ctx, towerID, domainStatus); err != nil {
		return err
	}

	if !eventNeeded {
		return nil
	}

	event := &domain.Event{
		TowerID:    towerID,
		Type:       domain.EventTypeFailure,
		Severity:   domain.EventSeverity(severity),
		Message:    fmt.Sprintf("nagios: host %s state=%s output=%s", status.Hostname, status.State, status.PluginOutput),
		OccurredAt: status.LastStateChange,
	}
	return s.eventService.Create(ctx, event)
}

func mapNagiosState(state interfaces.HostState) (domain.TowerStatus, bool, string) {
	switch state {
	case interfaces.HostStateUp:
		return domain.TowerStatusOnline, false, ""
	case interfaces.HostStateDown, interfaces.HostStateUnreachable:
		return domain.TowerStatusOffline, true, "critical"
	case interfaces.HostStatePending:
		return domain.TowerStatusDegraded, false, ""
	default:
		return domain.TowerStatusDegraded, true, "warning"
	}
}