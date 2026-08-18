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

// BackhaulScheduler polls towers for backhaul interface metrics (IF-MIB
// style) and converts them into domain.BackhaulInterface records.
type BackhaulScheduler struct {
	towersRepo interfaces.TowerRepository
	svc        *services.BackhaulInterfaceService
	collector  snmp.Collector
	profiles   map[string]snmp.Profile
	log        *zap.Logger
	interval   time.Duration
	batchSize  int
}

func NewBackhaulScheduler(
	towersRepo interfaces.TowerRepository,
	svc *services.BackhaulInterfaceService,
	collector snmp.Collector,
	profiles map[string]snmp.Profile,
	log *zap.Logger,
	interval time.Duration,
	batchSize int,
) *BackhaulScheduler {
	if interval <= 0 {
		interval = 60 * time.Second
	}
	if batchSize <= 0 {
		batchSize = 100
	}
	return &BackhaulScheduler{
		towersRepo: towersRepo,
		svc:        svc,
		collector:  collector,
		profiles:   profiles,
		log:        log.With(zap.String("component", "backhaul_scheduler")),
		interval:   interval,
		batchSize:  batchSize,
	}
}

func (s *BackhaulScheduler) Start(ctx context.Context) {
	s.log.Info("backhaul scheduler started", zap.String("interval", s.interval.String()))
	t := time.NewTicker(s.interval)
	defer t.Stop()

	s.CollectOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			s.log.Info("backhaul scheduler stopped")
			return
		case <-t.C:
			s.CollectOnce(ctx)
		}
	}
}

func (s *BackhaulScheduler) CollectOnce(ctx context.Context) {
	for offset := 0; ; {
		towers, total, err := s.towersRepo.List(ctx, interfaces.TowerFilter{Limit: s.batchSize, Offset: offset})
		if err != nil {
			s.log.Error("backhaul scheduler list towers failed", zap.Error(err), zap.Int("offset", offset))
			return
		}

		for _, tw := range towers {
			if !tw.SNMPEnabled {
				continue
			}
			vendor := tw.Vendor
			profile, ok := s.profiles[vendor]
			if !ok {
				s.log.Debug("backhaul scheduler no profile for vendor, skipping", zap.String("tower_id", tw.ID), zap.String("vendor", vendor))
				continue
			}

			samples, err := s.collector.Collect(ctx, tw, profile)
			if err != nil {
				s.log.Error("backhaul scheduler collect failed", zap.String("tower_id", tw.ID), zap.Error(err))
				continue
			}

			normalized := snmp.NormalizeSamples(profile, samples)
			if len(normalized) == 0 {
				s.log.Debug("backhaul scheduler no normalized metrics from samples", zap.String("tower_id", tw.ID))
				continue
			}

			ifaces := convertSamplesToBackhaulInterfaces(tw.ID, normalized)
			if len(ifaces) == 0 {
				s.log.Debug("backhaul scheduler no interfaces mapped from samples", zap.String("tower_id", tw.ID))
				continue
			}

			if err := s.svc.CollectMultipleInterfaces(ctx, ifaces); err != nil {
				s.log.Error("backhaul scheduler ingest failed", zap.String("tower_id", tw.ID), zap.Error(err))
				continue
			}

			s.log.Info("backhaul scheduler collected", zap.String("tower_id", tw.ID), zap.Int("interfaces", len(ifaces)))
		}

		offset += len(towers)
		if len(towers) == 0 || offset >= total {
			return
		}
	}
}

func convertSamplesToBackhaulInterfaces(towerIDStr string, samples map[string]float64) []*domain.BackhaulInterface {
	// Very small heuristic: if we have ifInOctets/ifOutOctets keys create a single
	// interface record. Extend for multiple interfaces per tower as needed.
	var any bool
	bi := &domain.BackhaulInterface{}
	if id, err := uuid.Parse(towerIDStr); err == nil {
		bi.TowerID = id
	}
	now := time.Now().UTC()
	bi.MeasuredAt = now
	bi.ReceivedAt = now
	bi.SourcePoller = "snmp_poller"

	if v, ok := samples["if_admin_status"]; ok {
		if v == 1 {
			bi.AdminStatus = "up"
		} else {
			bi.AdminStatus = "down"
		}
		any = true
	}
	if v, ok := samples["if_oper_status"]; ok {
		if v == 1 {
			bi.OperStatus = "up"
		} else {
			bi.OperStatus = "down"
		}
		any = true
	}
	if v, ok := samples["if_in_octets"]; ok {
		val := uint64(v)
		bi.InOctets = &val
		any = true
	}
	if v, ok := samples["if_out_octets"]; ok {
		val := uint64(v)
		bi.OutOctets = &val
		any = true
	}

	if !any {
		return nil
	}
	// Ensure an InterfaceID and Name so repository validations pass.
	if bi.InterfaceID == "" {
		bi.InterfaceID = uuid.New().String()
	}
	if bi.Name == "" {
		bi.Name = "if0"
	}

	return []*domain.BackhaulInterface{bi}
}
