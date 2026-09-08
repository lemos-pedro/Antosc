package services

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"unicode"

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

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// alarmTypeKey deriva uma chave estável a partir da descrição do alarme.
// "Compressor Fault", "compressor fault", "Compressor  Fault!!" → "compressor_fault"
//
// NÃO usar AlarmNo como identidade principal: no NetEco/Huawei o AlarmNo
// costuma ser único por ocorrência (sequência). Cada reenvio do mesmo
// problema gerava um evento/ticket novo (ex.: 73× Compressor Fault em
// KIMPAVITA). A descrição normalizada agrupa ocorrências do mesmo tipo.
func alarmTypeKey(description, alarmNo string) string {
	s := strings.ToLower(strings.TrimSpace(description))
	if s == "" || strings.Contains(s, "sem descrição") {
		// Sem descrição útil — fallback para AlarmNo (pior, mas evita
		// colidir todos os alarmes sem texto num único bucket).
		s = strings.ToLower(strings.TrimSpace(alarmNo))
		if s == "" {
			return "unknown"
		}
	}

	s = strings.Map(func(r rune) rune {
		switch r {
		case 'á', 'à', 'ã', 'â', 'ä':
			return 'a'
		case 'é', 'è', 'ê', 'ë':
			return 'e'
		case 'í', 'ì', 'î', 'ï':
			return 'i'
		case 'ó', 'ò', 'ô', 'õ', 'ö':
			return 'o'
		case 'ú', 'ù', 'û', 'ü':
			return 'u'
		case 'ç':
			return 'c'
		case 'ñ':
			return 'n'
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return ' '
	}, s)

	s = nonAlnum.ReplaceAllString(strings.TrimSpace(s), "_")
	s = strings.Trim(s, "_")
	if len(s) > 80 {
		s = s[:80]
	}
	if s == "" {
		return "unknown"
	}
	return s
}

// buildNetEcoAlarmKey produz a alarm_key estável usada em CreateOrTouch.
// Formato: neteco:{NEID}:{tipo_normalizado}
func buildNetEcoAlarmKey(neid, description, alarmNo string) string {
	return fmt.Sprintf("neteco:%s:%s", strings.TrimSpace(neid), alarmTypeKey(description, alarmNo))
}

// HandleTrap processa um alarme recebido via SNMP Trap do NetEco/Huawei.
//
// Regras:
//   - EventType 2 = clear → Resolve pelo mesmo alarm_key estável
//   - severity info / "Sem descrição" → ignorados (não criam evento nem ticket)
//   - CreateOrTouch com chave por tipo de alarme (não por AlarmNo de ocorrência)
//   - Ticket só na transição para evento NOVO
func (s *NetEcoAlarmService) HandleTrap(
	ctx context.Context,
	trap neteco.TrapEvent,
) {
	log.Printf(
		"[NetEco][alarm] processando trap site=%s neid=%s alarm=%s eventType=%d severity=%d source=%s desc=%q",
		trap.SiteName,
		trap.NEID,
		trap.AlarmNo,
		trap.EventType,
		trap.Severity,
		trap.SourceIP,
		trap.Description,
	)

	if strings.TrimSpace(trap.NEID) == "" {
		log.Printf("[NetEco][alarm] trap rejeitado: NEID vazio")
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

	message := strings.TrimSpace(trap.Description)
	if message == "" {
		if strings.TrimSpace(trap.AlarmNo) != "" {
			message = fmt.Sprintf("Alarme NetEco %s", trap.AlarmNo)
		} else {
			message = "Alarme NetEco sem descrição"
		}
	}

	alarmKey := buildNetEcoAlarmKey(trap.NEID, message, trap.AlarmNo)

	// EventType 2 = clear (alarme resolvido no NetEco).
	if trap.EventType == 2 {
		log.Printf("[NetEco][alarm] resolvendo evento key=%s towerID=%s", alarmKey, towerID)

		if err := s.eventSvc.Resolve(ctx, towerID, alarmKey); err != nil {
			log.Printf("[NetEco][alarm] erro ao resolver evento key=%s err=%v", alarmKey, err)
			return
		}

		log.Printf("[NetEco][alarm] evento resolvido key=%s", alarmKey)
		return
	}

	severity := mapSeverity(trap.Severity)

	// Ruído operacional: não abrir evento/ticket para info nem para
	// "Sem descrição do evento associado" (flood observado em KIMPAVITA).
	if severity == domain.EventSeverityInfo ||
		strings.Contains(strings.ToLower(message), "sem descrição") {
		log.Printf(
			"[NetEco][alarm] ignorado (info/sem descrição) key=%s message=%q",
			alarmKey, message,
		)
		return
	}

	event := &domain.Event{
		TowerID:    towerID,
		Type:       domain.EventTypeAlarm,
		Severity:   severity,
		Message:    message,
		DataSource: "neteco_proxy",
	}

	log.Printf(
		"[NetEco][alarm] persistindo evento towerID=%s key=%s severity=%s message=%q",
		towerID, alarmKey, severity, message,
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

		// Só abre ticket na transição para um evento novo, não em cada
		// re-toque de um alarme já ativo (mesma regra do SNMPIngestService).
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
// Escala típica NetEco: 1–2 critical, 3–4 warning, 5 info.
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
