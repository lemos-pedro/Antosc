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

// RadioScheduler polls towers by SNMP for radio-related OIDs and converts
// the collected samples into `domain.RadioKPI` batches, delegating to the
// RadioKPIService. Mapping is intentionally conservative: when expected
// keys are not present the sample is skipped.
type RadioScheduler struct {
	towersRepo interfaces.TowerRepository
	svc        *services.RadioKPIService
	collector  snmp.Collector
	profiles   map[string]snmp.Profile
	log        *zap.Logger
	interval   time.Duration
	batchSize  int
}

func NewRadioScheduler(
	towersRepo interfaces.TowerRepository,
	svc *services.RadioKPIService,
	collector snmp.Collector,
	profiles map[string]snmp.Profile,
	log *zap.Logger,
	interval time.Duration,
	batchSize int,
) *RadioScheduler {
	if interval <= 0 {
		interval = 60 * time.Second
	}
	if batchSize <= 0 {
		batchSize = 100
	}
	return &RadioScheduler{
		towersRepo: towersRepo,
		svc:        svc,
		collector:  collector,
		profiles:   profiles,
		log:        log.With(zap.String("component", "radio_scheduler")),
		interval:   interval,
		batchSize:  batchSize,
	}
}

func (s *RadioScheduler) Start(ctx context.Context) {
	s.log.Info("radio scheduler started", zap.String("interval", s.interval.String()))
	t := time.NewTicker(s.interval)
	defer t.Stop()

	s.CollectOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			s.log.Info("radio scheduler stopped")
			return
		case <-t.C:
			s.CollectOnce(ctx)
		}
	}
}

func (s *RadioScheduler) CollectOnce(ctx context.Context) {
	for offset := 0; ; {
		towers, total, err := s.towersRepo.List(ctx, interfaces.TowerFilter{Limit: s.batchSize, Offset: offset})
		if err != nil {
			s.log.Error("radio scheduler list towers failed", zap.Error(err), zap.Int("offset", offset))
			return
		}

		for _, tw := range towers {
			if !tw.SNMPEnabled {
				continue
			}
			vendor := tw.Vendor
			profile, ok := s.profiles[vendor]
			if !ok {
				s.log.Debug("radio scheduler no profile for vendor, skipping", zap.String("tower_id", tw.ID), zap.String("vendor", vendor))
				continue
			}

			samples, err := s.collector.Collect(ctx, tw, profile)
			if err != nil {
				s.log.Error("radio scheduler collect failed", zap.String("tower_id", tw.ID), zap.Error(err))
				continue
			}

			normalized := snmp.NormalizeSamples(profile, samples)
			if len(normalized) == 0 {
				s.log.Debug("radio scheduler no normalized metrics from samples", zap.String("tower_id", tw.ID))
				continue
			}

			kpis := convertSamplesToRadioKPIs(tw.ID, normalized)
			if len(kpis) == 0 {
				s.log.Debug("radio scheduler no radio KPIs mapped from samples", zap.String("tower_id", tw.ID))
				continue
			}

			if err := s.svc.IngestKPIs(ctx, kpis); err != nil {
				s.log.Error("radio scheduler ingest KPIs failed", zap.String("tower_id", tw.ID), zap.Error(err))
				continue
			}

			s.log.Info("radio scheduler collected", zap.String("tower_id", tw.ID), zap.Int("kpis", len(kpis)))
		}

		offset += len(towers)
		if len(towers) == 0 || offset >= total {
			return
		}
	}
}

// convertSamplesToRadioKPIs creates a basic RadioKPI from raw float samples.
// This is intentionally simple: it looks for common keys and fills the struct
// where available. Extend as needed for vendor-specific mappings.
func convertSamplesToRadioKPIs(towerIDStr string, samples map[string]float64) []*domain.RadioKPI {
	var any bool
	k := &domain.RadioKPI{}
	// parse tower id
	if id, err := uuid.Parse(towerIDStr); err == nil {
		k.TowerID = id
	}
	now := time.Now().UTC()
	k.MeasuredAt = now
	k.ReceivedAt = now
	k.SourceSystem = "snmp_poller"
	k.CollectionIntervalSec = int(0)

	if v, ok := samples["tx_power_dbm"]; ok {
		any = true
		k.TxPowerDbm = &v
	}
	if v, ok := samples["rx_power_dbm"]; ok {
		any = true
		k.RxPowerDbm = &v
	}
	if v, ok := samples["snr_db"]; ok {
		any = true
		k.SnrDb = &v
	}
	if v, ok := samples["sinr_db"]; ok {
		any = true
		k.SinrDb = &v
	}
	if v, ok := samples["rsrp_dbm"]; ok {
		any = true
		k.RsrpDbm = &v
	}
	if v, ok := samples["rsrq_db"]; ok {
		any = true
		k.RsrqDbm = &v
	}

	if v64, ok := samples["connected_ues"]; ok {
		v := int(v64)
		k.ConnectedUEs = &v
		any = true
	}

	if !any {
		return nil
	}
	return []*domain.RadioKPI{k}
}
