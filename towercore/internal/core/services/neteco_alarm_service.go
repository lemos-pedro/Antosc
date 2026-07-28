package services

import (
	"context"
	"fmt"
	"log"
	"strings"

	"towercore/internal/adapters/neteco"
	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
)

type NetEcoAlarmService struct {
	eventSvc      *EventService
	towerRepo     interfaces.TowerRepository
	ticketService *TicketService
}

func NewNetEcoAlarmService(
	eventSvc *EventService,
	towerRepo interfaces.TowerRepository,
	ticketService *TicketService,
) *NetEcoAlarmService {
	return &NetEcoAlarmService{
		eventSvc:      eventSvc,
		towerRepo:     towerRepo,
		ticketService: ticketService,
	}
}

// HandleTrap processa um alarme recebido via SNMP Trap.
func (s *NetEcoAlarmService) HandleTrap(
	ctx context.Context,
	trap neteco.TrapEvent,
) {
	log.Printf(
		"[NetEco][alarm] processando trap site=%s neid=%s alarm=%s eventType=%d severity=%d source=%s",
		trap.SiteName,
		trap.NEID,
		trap.AlarmNo,
		trap.EventType,
		trap.Severity,
		trap.SourceIP,
	)

	if strings.TrimSpace(trap.NEID) == "" {
		log.Printf("[NetEco][alarm] trap rejeitado: NEID vazio")
		return
	}

	if strings.TrimSpace(trap.AlarmNo) == "" {
		log.Printf("[NetEco][alarm] trap rejeitado: AlarmNo vazio NEID=%s", trap.NEID)
		return
	}

	towerID, err := s.resolveTowerID(ctx, trap.NEID)
	if err != nil {
		log.Printf(
			"[NetEco][alarm] torre não encontrada NEID=%s site=%s err=%v",
			trap.NEID, trap.SiteName, err,
		)
		return
	}

	log.Printf("[NetEco][alarm] torre encontrada towerID=%s NEID=%s", towerID, trap.NEID)

	alarmKey := fmt.Sprintf("neteco:%s:%s", trap.NEID, trap.AlarmNo)

	// EventType 2 = clear.
	if trap.EventType == 2 {
		log.Printf("[NetEco][alarm] resolvendo evento key=%s towerID=%s", alarmKey, towerID)

		if err := s.eventSvc.Resolve(ctx, towerID, alarmKey); err != nil {
			log.Printf("[NetEco][alarm] erro ao resolver evento key=%s err=%v", alarmKey, err)
			return
		}

		log.Printf("[NetEco][alarm] evento resolvido key=%s", alarmKey)
		return
	}

	message := strings.TrimSpace(trap.Description)
	if message == "" {
		message = fmt.Sprintf("Alarme NetEco %s", trap.AlarmNo)
	}

	event := &domain.Event{
		TowerID:    towerID,
		Type:       domain.EventTypeAlarm,
		Severity:   mapSeverity(trap.Severity),
		Message:    message,
		DataSource: "neteco_proxy",
	}

	log.Printf(
		"[NetEco][alarm] persistindo evento towerID=%s key=%s severity=%d message=%q",
		towerID, alarmKey, trap.Severity, message,
	)

	createdEvent, created, err := s.eventSvc.CreateOrTouch(ctx, event, alarmKey)
	if err != nil {
		log.Printf(
			"[NetEco][alarm] erro ao criar/tocar evento key=%s towerID=%s err=%v",
			alarmKey, towerID, err,
		)
		return
	}

	if created {
		log.Printf(
			"[NetEco][alarm] NOVO EVENTO CRIADO site=%s towerID=%s key=%s message=%q",
			trap.SiteName, towerID, alarmKey, message,
		)

		// Mesma regra que o SNMPIngestService: só abre ticket na transição
		// para um evento novo, não em cada re-toque de um alarme já ativo.
		// Sem isto, alarmes NetEco nunca aparecem na UI de "Alarmes"
		// (alarms-store.tsx lê tickets, não events diretamente).
		if s.ticketService != nil {
			if _, err := s.ticketService.Create(ctx, towerID, createdEvent.ID); err != nil {
				log.Printf(
					"[NetEco][alarm] erro ao criar ticket key=%s towerID=%s err=%v",
					alarmKey, towerID, err,
				)
			}
		}

		return
	}

	log.Printf("[NetEco][alarm] evento existente atualizado key=%s towerID=%s", alarmKey, towerID)
}

// resolveTowerID encontra a torre pelo NEID configurado no NetEco.
func (s *NetEcoAlarmService) resolveTowerID(ctx context.Context, neid string) (string, error) {
	neid = strings.TrimSpace(neid)
	if neid == "" {
		return "", fmt.Errorf("NEID vazio")
	}

	enabled := true
	towers, _, err := s.towerRepo.List(ctx, interfaces.TowerFilter{
		NetecoEnabled: &enabled,
		Limit:         500,
	})
	if err != nil {
		return "", fmt.Errorf("listar torres NetEco: %w", err)
	}

	for _, tower := range towers {
		if strings.TrimSpace(tower.NetecoNEID) == neid {
			return tower.ID, nil
		}
	}

	return "", fmt.Errorf("nenhuma torre com neteco_neid=%s", neid)
}

// mapSeverity converte severidade Huawei para o domínio.
func mapSeverity(huaweiSeverity int) domain.EventSeverity {
	switch huaweiSeverity {
	case 1:
		return domain.EventSeverityCritical
	case 2:
		return domain.EventSeverityCritical
	case 3:
		return domain.EventSeverityWarning
	case 4:
		return domain.EventSeverityWarning
	case 5:
		return domain.EventSeverityInfo
	default:
		return domain.EventSeverityInfo
	}
}