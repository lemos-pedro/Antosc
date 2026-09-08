package scheduler

import (
	"context"
	"strings"
	"time"

	"towercore/internal/core/interfaces"
	"towercore/internal/core/services"
	"towercore/internal/infrastructure/logger"
)

// NagiosPoller é o port mínimo que o scheduler precisa para consultar
// o estado de um host no Nagios. Implementado por nagios.Client
// (internal/adapters/nagios/client.go).
type NagiosPoller interface {
	FetchHostStatus(ctx context.Context, hostname string) (interfaces.HostStatus, error)
}

type NagiosScheduler struct {
	towersRepo    interfaces.TowerRepository
	ingestService *services.NagiosIngestService
	poller        NagiosPoller
	log           *logger.Logger
	interval      time.Duration
	batchSize     int
}

func NewNagiosScheduler(
	towersRepo interfaces.TowerRepository,
	ingestService *services.NagiosIngestService,
	poller NagiosPoller,
	log *logger.Logger,
	interval time.Duration,
	batchSize int,
) *NagiosScheduler {
	if interval <= 0 {
		interval = 60 * time.Second
	}
	if batchSize <= 0 {
		batchSize = 100
	}
	return &NagiosScheduler{
		towersRepo:    towersRepo,
		ingestService: ingestService,
		poller:        poller,
		log:           log,
		interval:      interval,
		batchSize:     batchSize,
	}
}

func (s *NagiosScheduler) Start(ctx context.Context) {
	s.log.Infof("nagios scheduler started interval=%s batch=%d", s.interval.String(), s.batchSize)
	t := time.NewTicker(s.interval)
	defer t.Stop()

	s.CollectOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			s.log.Info("nagios scheduler stopped")
			return
		case <-t.C:
			s.CollectOnce(ctx)
		}
	}
}

func (s *NagiosScheduler) CollectOnce(ctx context.Context) {
	for offset := 0; ; {
		towers, total, err := s.towersRepo.List(ctx, interfaces.TowerFilter{
			Limit:  s.batchSize,
			Offset: offset,
		})
		if err != nil {
			s.log.Errorf("nagios scheduler list towers failed: %v", err)
			return
		}

		for _, tw := range towers {
			if !tw.NagiosEnabled {
				continue
			}

			hostname := strings.TrimSpace(tw.NagiosHostname)
			if hostname == "" {
				s.log.Errorf("nagios scheduler skipped tower=%s reason=missing nagios_hostname", tw.ID)
				continue
			}

			status, err := s.poller.FetchHostStatus(ctx, hostname)
			if err != nil {
				s.log.Errorf("nagios scheduler fetch failed tower=%s hostname=%s err=%v", tw.ID, hostname, err)
				// Sem isto, a torre ficava presa no último status bom
				// quando perdíamos comunicação com o Nagios — mesmo bug
				// que já tinha sido corrigido no SNMP scheduler.
				if markErr := s.ingestService.MarkUnreachable(ctx, tw.ID); markErr != nil {
					s.log.Errorf("nagios scheduler mark unreachable failed tower=%s err=%v", tw.ID, markErr)
				}
				continue
			}

			if err := s.ingestService.Ingest(ctx, tw.ID, status); err != nil {
				s.log.Errorf("nagios scheduler ingest failed tower=%s hostname=%s err=%v", tw.ID, hostname, err)
				continue
			}

			s.log.Infof("nagios scheduler collected tower=%s hostname=%s state=%s", tw.ID, hostname, status.State)
		}

		offset += len(towers)
		if len(towers) == 0 || offset >= total {
			return
		}
	}
}
