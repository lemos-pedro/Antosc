package services

import (
	"context"
	"fmt"
	"time"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
	"towercore/internal/infrastructure/logger"
)

// Thresholds default para geração de eventos. Podem vir a ser configuráveis
// por link no futuro (ex: link crítico do PCA com threshold mais apertado),
// mas fixos aqui para a primeira versão — mesmo princípio "sem valores
// inventados": estes são limites explícitos e documentados, não silenciosos.
const (
	HighUtilizationThresholdPercent = 85.0
	ErrorRateWarningThreshold       = 100 // erros acumulados entre polls
	ErrorRateCriticalThreshold      = 1000
)

// LinkStatusService aplica as regras de negócio sobre métricas recolhidas
// pelo adapter: calcula bps/utilização por delta de contadores, decide
// status e gera/atualiza eventos via CreateOrTouch (mesmo padrão de
// deduplicação já usado para alarmes de torre).
type LinkStatusService struct {
	linkRepo  interfaces.NetworkLinkRepository
	eventRepo interfaces.NetworkLinkEventRepository
	monitor   interfaces.LinkMonitor
	log       *logger.Logger
}

func NewLinkStatusService(
	linkRepo interfaces.NetworkLinkRepository,
	eventRepo interfaces.NetworkLinkEventRepository,
	monitor interfaces.LinkMonitor,
	log *logger.Logger,
) *LinkStatusService {
	return &LinkStatusService{
		linkRepo:  linkRepo,
		eventRepo: eventRepo,
		monitor:   monitor,
		log:       log,
	}
}

// EvaluateLink recolhe métricas atuais de um link, calcula utilização por
// comparação com o snapshot anterior, persiste, e gera eventos conforme
// necessário. É esta função que o scheduler chama por cada link, por ciclo.
func (s *LinkStatusService) EvaluateLink(ctx context.Context, link *domain.NetworkLink) error {
	previous, err := s.linkRepo.GetLastSnapshot(ctx, link.LinkID)
	if err != nil {
		// Sem snapshot anterior (primeira leitura) não é erro fatal —
		// apenas não há delta para calcular bps ainda.
		s.log.Infof("link_status_service: sem snapshot anterior para link %s (primeira leitura)", link.LinkID)
	}

	current, err := s.monitor.GetInterfaceMetrics(ctx, link.RouterIP, link.IfIndex)
	if err != nil {
		return fmt.Errorf("link_status_service: falha ao recolher métricas do link %s: %w", link.LinkID, err)
	}
	current.LinkID = link.LinkID

	if previous != nil {
		s.calculateUtilization(current, previous, link.NominalCapacityMb)
	}

	if err := s.linkRepo.SaveMetricSnapshot(ctx, current); err != nil {
		return fmt.Errorf("link_status_service: falha ao gravar snapshot do link %s: %w", link.LinkID, err)
	}

	return s.evaluateEvents(ctx, link, current, previous)
}

// calculateUtilization deriva bps de entrada/saída a partir do delta de
// contadores entre dois snapshots, e converte para percentagem da
// capacidade nominal contratada. Assume contadores HC (64-bit) — ver nota
// em profile.go sobre overflow em interfaces de alta capacidade.
func (s *LinkStatusService) calculateUtilization(current, previous *domain.LinkMetricSnapshot, nominalCapacityMb int64) {
	elapsedSeconds := current.CollectedAt.Sub(previous.CollectedAt).Seconds()
	if elapsedSeconds <= 0 || nominalCapacityMb <= 0 {
		return
	}

	// Contador pode ter reiniciado (reboot do router) — se o valor atual
	// for menor que o anterior, não calculamos delta negativo.
	if current.InOctets >= previous.InOctets {
		deltaInBits := float64(current.InOctets-previous.InOctets) * 8
		inBps := deltaInBits / elapsedSeconds
		current.InUtilPercent = (inBps / (float64(nominalCapacityMb) * 1_000_000)) * 100
	}
	if current.OutOctets >= previous.OutOctets {
		deltaOutBits := float64(current.OutOctets-previous.OutOctets) * 8
		outBps := deltaOutBits / elapsedSeconds
		current.OutUtilPercent = (outBps / (float64(nominalCapacityMb) * 1_000_000)) * 100
	}
}

func (s *LinkStatusService) evaluateEvents(ctx context.Context, link *domain.NetworkLink, current, previous *domain.LinkMetricSnapshot) error {
	now := time.Now().UTC()

	// Regra 1: link down
	downKey := fmt.Sprintf("link_down:%s", link.LinkID)
	if current.OperStatus == domain.LinkStatusDown || current.OperStatus == domain.LinkStatusLowerLayerDown {
		event := &domain.LinkEvent{
			LinkID:     link.LinkID,
			Type:       "down",
			Severity:   "critical",
			Message:    fmt.Sprintf("Link %s (%s) reporta operStatus=%s", link.Name, link.IfAlias, current.OperStatus),
			OccurredAt: now,
			AlarmKey:   downKey,
		}
		if err := s.eventRepo.CreateOrTouch(ctx, event); err != nil {
			return fmt.Errorf("link_status_service: falha ao registar evento down para %s: %w", link.LinkID, err)
		}
	} else {
		if err := s.eventRepo.Resolve(ctx, downKey); err != nil {
			s.log.Errorf("link_status_service: falha ao resolver evento down para %s: %v", link.LinkID, err)
		}
	}

	// Regra 2: utilização alta (só faz sentido com snapshot anterior disponível)
	utilKey := fmt.Sprintf("link_high_util:%s", link.LinkID)
	if previous != nil && (current.InUtilPercent >= HighUtilizationThresholdPercent || current.OutUtilPercent >= HighUtilizationThresholdPercent) {
		event := &domain.LinkEvent{
			LinkID:     link.LinkID,
			Type:       "high_utilization",
			Severity:   "warning",
			Message:    fmt.Sprintf("Link %s a >=%.0f%% de utilização (in=%.1f%%, out=%.1f%%)", link.Name, HighUtilizationThresholdPercent, current.InUtilPercent, current.OutUtilPercent),
			OccurredAt: now,
			AlarmKey:   utilKey,
		}
		if err := s.eventRepo.CreateOrTouch(ctx, event); err != nil {
			return fmt.Errorf("link_status_service: falha ao registar evento de utilização para %s: %w", link.LinkID, err)
		}
	} else {
		if err := s.eventRepo.Resolve(ctx, utilKey); err != nil {
			s.log.Errorf("link_status_service: falha ao resolver evento de utilização para %s: %v", link.LinkID, err)
		}
	}

	// Regra 3: taxa de erros (indicador de qualidade — CRC errors, discards)
	errorKey := fmt.Sprintf("link_errors:%s", link.LinkID)
	totalErrors := current.InErrors + current.OutErrors
	switch {
	case totalErrors >= ErrorRateCriticalThreshold:
		event := &domain.LinkEvent{
			LinkID:     link.LinkID,
			Type:       "high_errors",
			Severity:   "critical",
			Message:    fmt.Sprintf("Link %s com %d erros acumulados (in=%d, out=%d) — possível degradação física", link.Name, totalErrors, current.InErrors, current.OutErrors),
			OccurredAt: now,
			AlarmKey:   errorKey,
		}
		if err := s.eventRepo.CreateOrTouch(ctx, event); err != nil {
			return fmt.Errorf("link_status_service: falha ao registar evento de erros para %s: %w", link.LinkID, err)
		}
	case totalErrors >= ErrorRateWarningThreshold:
		event := &domain.LinkEvent{
			LinkID:     link.LinkID,
			Type:       "high_errors",
			Severity:   "warning",
			Message:    fmt.Sprintf("Link %s com %d erros acumulados (in=%d, out=%d)", link.Name, totalErrors, current.InErrors, current.OutErrors),
			OccurredAt: now,
			AlarmKey:   errorKey,
		}
		if err := s.eventRepo.CreateOrTouch(ctx, event); err != nil {
			return fmt.Errorf("link_status_service: falha ao registar evento de erros para %s: %w", link.LinkID, err)
		}
	default:
		if err := s.eventRepo.Resolve(ctx, errorKey); err != nil {
			s.log.Errorf("link_status_service: falha ao resolver evento de erros para %s: %v", link.LinkID, err)
		}
	}

	return nil
}
