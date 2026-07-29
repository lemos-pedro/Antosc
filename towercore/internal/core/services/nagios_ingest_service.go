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

	alarmKey := "nagios_host_status"

	if !eventNeeded {
		// Estado saudável ou neutro: resolve qualquer evento nagios aberto.
		return s.eventService.Resolve(ctx, towerID, alarmKey)
	}

	event := &domain.Event{
		TowerID:    towerID,
		Type:       domain.EventTypeFailure,
		Severity:   domain.EventSeverity(severity),
		Message:    fmt.Sprintf("nagios: host %s state=%s output=%s", status.Hostname, status.State, status.PluginOutput),
		OccurredAt: status.LastStateChange,
	}
	_, _, err := s.eventService.CreateOrTouch(ctx, event, alarmKey)
	return err
}

// MarkUnreachable marca a torre como offline quando a própria consulta ao
// Nagios falha (timeout de rede, CGI indisponível, hostname não resolvido).
// Antes desta função existir, uma falha de fetch era apenas registada em
// log e ignorada (ver nagios_scheduler.go) — a torre ficava presa no
// último status bom, mesmo sem qualquer comunicação real. Segue o mesmo
// padrão de SNMPIngestService.MarkUnreachable.
func (s *NagiosIngestService) MarkUnreachable(ctx context.Context, towerID string) error {
	if strings.TrimSpace(towerID) == "" {
		return errors.New("tower_id is required")
	}
	if s.towerUpdater == nil {
		return nil
	}
	return s.towerUpdater.UpdateStatus(ctx, towerID, domain.TowerStatusOffline)
}

func mapNagiosState(state interfaces.HostState) (domain.TowerStatus, bool, string) {
	switch state {
	case interfaces.HostStateUp:
		return domain.TowerStatusOnline, false, ""
	case interfaces.HostStateDown, interfaces.HostStateUnreachable:
		return domain.TowerStatusOffline, true, "critical"
	case interfaces.HostStatePending:
		return domain.TowerStatusDegraded, true, "warning"
	default:
		return domain.TowerStatusDegraded, true, "warning"
	}
}