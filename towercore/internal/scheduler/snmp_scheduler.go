package scheduler

import (
	"context"
	"strings"
	"time"

	"towercore/internal/adapters/snmp"
	"towercore/internal/core/interfaces"
	"towercore/internal/core/services"
	"towercore/internal/infrastructure/logger"
)

type SNMPScheduler struct {
	towersRepo    interfaces.TowerRepository
	ingestService *services.SNMPIngestService
	collector     snmp.Collector
	profiles      map[string]snmp.Profile
	log           *logger.Logger
	interval      time.Duration
	batchSize     int
}

func NewSNMPScheduler(
	towersRepo interfaces.TowerRepository,
	ingestService *services.SNMPIngestService,
	collector snmp.Collector,
	profiles map[string]snmp.Profile,
	log *logger.Logger,
	interval time.Duration,
	batchSize int,
) *SNMPScheduler {
	if interval <= 0 {
		interval = 60 * time.Second
	}
	if batchSize <= 0 {
		batchSize = 100
	}
	return &SNMPScheduler{
		towersRepo:    towersRepo,
		ingestService: ingestService,
		collector:     collector,
		profiles:      profiles,
		log:           log,
		interval:      interval,
		batchSize:     batchSize,
	}
}

func (s *SNMPScheduler) Start(ctx context.Context) {
	s.log.Infof("snmp scheduler started interval=%s batch=%d", s.interval.String(), s.batchSize)
	t := time.NewTicker(s.interval)
	defer t.Stop()

	s.CollectOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			s.log.Info("snmp scheduler stopped")
			return
		case <-t.C:
			s.CollectOnce(ctx)
		}
	}
}

func (s *SNMPScheduler) CollectOnce(ctx context.Context) {
	for offset := 0; ; {
		towers, total, err := s.towersRepo.List(ctx, interfaces.TowerFilter{
			Limit:  s.batchSize,
			Offset: offset,
		})
		if err != nil {
			s.log.Errorf("snmp scheduler list towers failed: %v", err)
			return
		}

		for _, tw := range towers {
			if !tw.SNMPEnabled {
				continue
			}

			vendor := strings.ToLower(strings.TrimSpace(tw.Vendor))
			profile, ok := s.profiles[vendor]
			if !ok {
				s.log.Errorf("snmp scheduler skipped tower=%s reason=unsupported vendor=%s", tw.ID, tw.Vendor)
				continue
			}

			samples, err := s.collector.Collect(ctx, tw, profile)
			if err != nil {
				s.log.Errorf("snmp scheduler collect failed tower=%s vendor=%s err=%v", tw.ID, vendor, err)
				if markErr := s.ingestService.MarkUnreachable(ctx, tw.ID); markErr != nil {
					s.log.Errorf("snmp scheduler mark unreachable failed tower=%s err=%v", tw.ID, markErr)
				}
				continue
			}

			err = s.ingestService.Ingest(ctx, services.SNMPSnapshot{
				TowerID:     tw.ID,
				Vendor:      vendor,
				CollectedAt: time.Now().UTC(),
				Samples:     samples,
			})
			if err != nil {
				s.log.Errorf("snmp scheduler ingest failed tower=%s vendor=%s err=%v", tw.ID, vendor, err)
				continue
			}

			s.log.Infof("snmp scheduler collected tower=%s vendor=%s metrics=%d", tw.ID, vendor, len(samples))
		}

		offset += len(towers)
		if len(towers) == 0 || offset >= total {
			return
		}
	}
}
