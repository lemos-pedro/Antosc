package scheduler

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"towercore/internal/adapters/snmp"
	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
	"towercore/internal/core/services"
)

// SiteEnvironmentScheduler polls towers for environment sensors (temperature,
// humidity, door sensors, UPS status) and persists SiteEnvironment records.
type SiteEnvironmentScheduler struct {
	towersRepo interfaces.TowerRepository
	svc        *services.SiteEnvironmentService
	collector  snmp.Collector
	profiles   map[string]snmp.Profile
	log        *zap.Logger
	interval   time.Duration
	batchSize  int
}

func NewSiteEnvironmentScheduler(
	towersRepo interfaces.TowerRepository,
	svc *services.SiteEnvironmentService,
	collector snmp.Collector,
	profiles map[string]snmp.Profile,
	log *zap.Logger,
	interval time.Duration,
	batchSize int,
) *SiteEnvironmentScheduler {
	if interval <= 0 {
		interval = 60 * time.Second
	}
	if batchSize <= 0 {
		batchSize = 100
	}
	return &SiteEnvironmentScheduler{
		towersRepo: towersRepo,
		svc:        svc,
		collector:  collector,
		profiles:   profiles,
		log:        log.With(zap.String("component", "site_environment_scheduler")),
		interval:   interval,
		batchSize:  batchSize,
	}
}

func (s *SiteEnvironmentScheduler) Start(ctx context.Context) {
	s.log.Info("site environment scheduler started", zap.String("interval", s.interval.String()))
	t := time.NewTicker(s.interval)
	defer t.Stop()

	s.CollectOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			s.log.Info("site environment scheduler stopped")
			return
		case <-t.C:
			s.CollectOnce(ctx)
		}
	}
}

func (s *SiteEnvironmentScheduler) CollectOnce(ctx context.Context) {
	for offset := 0; ; {
		towers, total, err := s.towersRepo.List(ctx, interfaces.TowerFilter{Limit: s.batchSize, Offset: offset})
		if err != nil {
			s.log.Error("site env scheduler list towers failed", zap.Error(err), zap.Int("offset", offset))
			return
		}

		for _, tw := range towers {
			if !tw.SNMPEnabled {
				continue
			}
			vendor := tw.Vendor
			profile, ok := s.profiles[vendor]
			if !ok {
				s.log.Debug("site env scheduler no profile for vendor, skipping", zap.String("tower_id", tw.ID), zap.String("vendor", vendor))
				continue
			}

			samples, err := s.collector.Collect(ctx, tw, profile)
			if err != nil {
				s.log.Error("site env scheduler collect failed", zap.String("tower_id", tw.ID), zap.Error(err))
				continue
			}

			normalized := snmp.NormalizeSamples(profile, samples)
			if len(normalized) == 0 {
				s.log.Debug("site env scheduler no normalized metrics from samples", zap.String("tower_id", tw.ID))
				continue
			}

			envs := convertSamplesToSiteEnv(tw.ID, normalized)
			if len(envs) == 0 {
				s.log.Debug("site env scheduler no env mapped from samples", zap.String("tower_id", tw.ID))
				continue
			}

			if err := s.svc.CollectMultipleEnvironment(ctx, envs); err != nil {
				s.log.Error("site env scheduler ingest failed", zap.String("tower_id", tw.ID), zap.Error(err))
				continue
			}

			s.log.Info("site env scheduler collected", zap.String("tower_id", tw.ID), zap.Int("records", len(envs)))
		}

		offset += len(towers)
		if len(towers) == 0 || offset >= total {
			return
		}
	}
}

func convertSamplesToSiteEnv(towerIDStr string, samples map[string]float64) []*domain.SiteEnvironment {
	var any bool
	se := &domain.SiteEnvironment{}
	if id, err := uuid.Parse(towerIDStr); err == nil {
		se.SiteID = id
	}
	now := time.Now().UTC()
	se.MeasuredAt = now
	se.ReceivedAt = now
	se.SourcePoller = "snmp_poller"
	se.CollectionIntervalSec = 0

	if v, ok := samples["internal_temp_c"]; ok {
		any = true
		se.InternalTempC = &v
	}
	if v, ok := samples["humidity_pct"]; ok {
		any = true
		se.HumidityPct = &v
	}
	if v, ok := samples["mains_power_ok"]; ok {
		val := v == 1
		se.MainsPowerOk = &val
		any = true
	}
	if v, ok := samples["ups_on_battery"]; ok {
		val := v == 1
		se.UpsOnBattery = &val
		any = true
	}

	if !any {
		return nil
	}
	return []*domain.SiteEnvironment{se}
}
