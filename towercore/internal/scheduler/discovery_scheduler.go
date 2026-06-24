package scheduler

import (
	"context"
	"time"

	"towercore/internal/core/services"
	"towercore/internal/infrastructure/logger"
)

// DiscoveryScheduler corre periodicamente um network discovery SNMP
// sobre um CIDR fixo, registando os IPs que responderem em
// discovered_devices (via DiscoveryService). É deliberadamente
// paralelo e independente do SNMPScheduler: discovery encontra
// candidatos a torre; o SNMPScheduler sonda torres já promovidas e
// confirmadas em towers.
type DiscoveryScheduler struct {
	discovery *services.DiscoveryService
	cidr      string
	log       *logger.Logger
	interval  time.Duration
}

// NewDiscoveryScheduler cria o scheduler. interval default 10 minutos
// — discovery é mais pesado e menos urgente que o polling de métricas,
// por isso corre com um período bem mais longo que o SNMPScheduler.
func NewDiscoveryScheduler(
	discovery *services.DiscoveryService,
	cidr string,
	log *logger.Logger,
	interval time.Duration,
) *DiscoveryScheduler {
	if interval <= 0 {
		interval = 10 * time.Minute
	}
	return &DiscoveryScheduler{
		discovery: discovery,
		cidr:      cidr,
		log:       log,
		interval:  interval,
	}
}

func (s *DiscoveryScheduler) Start(ctx context.Context) {
	s.log.Infof("discovery scheduler started cidr=%s interval=%s", s.cidr, s.interval.String())
	t := time.NewTicker(s.interval)
	defer t.Stop()

	s.ScanOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			s.log.Info("discovery scheduler stopped")
			return
		case <-t.C:
			s.ScanOnce(ctx)
		}
	}
}

func (s *DiscoveryScheduler) ScanOnce(ctx context.Context) {
	result, err := s.discovery.ScanCIDR(ctx, s.cidr)
	if err != nil {
		s.log.Errorf("discovery scan failed cidr=%s err=%v", s.cidr, err)
		return
	}
	s.log.Infof(
		"discovery scan completed cidr=%s scanned=%d responded=%d errors=%d",
		s.cidr, result.Scanned, result.Responded, result.Errors,
	)
}
