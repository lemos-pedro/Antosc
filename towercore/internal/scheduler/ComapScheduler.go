package scheduler

import (
	"context"
	"time"

	"towercore/internal/adapters/comap"
	"towercore/internal/core/interfaces"
	"towercore/internal/core/services"
	"towercore/internal/infrastructure/logger"
)

// ComapScheduler faz polling periódico dos endpoints ComAp (gerador) por
// torre, seguindo exatamente o mesmo esqueleto do SNMPScheduler. A
// diferença principal: em vez de resolver o profile por vendor, resolve o
// endpoint (IP/porta/slave_id) via TowerEndpointRepository, filtrando por
// equipment_type="generator" e enabled=true.
type ComapScheduler struct {
	endpointsRepo interfaces.TowerEndpointRepository
	ingestService *services.ComapIngestService
	log           *logger.Logger
	interval      time.Duration
	batchSize     int
	modbusTimeout time.Duration
}

func NewComapScheduler(
	endpointsRepo interfaces.TowerEndpointRepository,
	ingestService *services.ComapIngestService,
	log *logger.Logger,
	interval time.Duration,
	batchSize int,
	modbusTimeout time.Duration,
) *ComapScheduler {
	if interval <= 0 {
		interval = 60 * time.Second
	}
	if batchSize <= 0 {
		batchSize = 100
	}
	if modbusTimeout <= 0 {
		modbusTimeout = 2 * time.Second
	}
	return &ComapScheduler{
		endpointsRepo: endpointsRepo,
		ingestService: ingestService,
		log:           log,
		interval:      interval,
		batchSize:     batchSize,
		modbusTimeout: modbusTimeout,
	}
}

func (s *ComapScheduler) Start(ctx context.Context) {
	s.log.Infof("comap scheduler started interval=%s batch=%d", s.interval.String(), s.batchSize)
	t := time.NewTicker(s.interval)
	defer t.Stop()

	s.CollectOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			s.log.Info("comap scheduler stopped")
			return
		case <-t.C:
			s.CollectOnce(ctx)
		}
	}
}

// CollectOnce percorre os endpoints ComAp habilitados e faz uma leitura.
// Qualquer falha (conexão Modbus ou leitura/gravação) marca a torre como
// unreachable — nunca deixa o estado anterior "preso" quando perdemos
// comunicação com o gerador.
func (s *ComapScheduler) CollectOnce(ctx context.Context) {
	endpoints, _, err := s.endpointsRepo.List(ctx, interfaces.TowerEndpointFilter{
		EquipmentType: "generator",
		Enabled:       boolPtr(true),
		Limit:         s.batchSize,
		Offset:        0,
	})
	if err != nil {
		s.log.Errorf("comap scheduler list endpoints failed: %v", err)
		return
	}

	for _, ep := range endpoints {
		if ep.IPAddress == "" {
			s.log.Errorf("comap scheduler skipped tower=%s reason=missing_ip", ep.TowerID)
			continue
		}

		client, err := comap.NewTCPClient(ep.IPAddress, ep.Port, uint8(ep.SlaveID), s.modbusTimeout)
		if err != nil {
			s.log.Errorf("comap scheduler connect failed tower=%s ip=%s err=%v", ep.TowerID, ep.IPAddress, err)
			if markErr := s.ingestService.MarkUnreachable(ctx, ep.TowerID); markErr != nil {
				s.log.Errorf("comap scheduler mark unreachable failed tower=%s err=%v", ep.TowerID, markErr)
			}
			continue
		}

		reader := comap.NewReader(client, uint8(ep.SlaveID))
		if err := s.ingestService.Ingest(ctx, ep.TowerID, reader); err != nil {
			s.log.Errorf("comap scheduler ingest failed tower=%s err=%v", ep.TowerID, err)
			if markErr := s.ingestService.MarkUnreachable(ctx, ep.TowerID); markErr != nil {
				s.log.Errorf("comap scheduler mark unreachable failed tower=%s err=%v", ep.TowerID, markErr)
			}
		} else {
			s.log.Infof("comap scheduler collected tower=%s ip=%s", ep.TowerID, ep.IPAddress)
		}

		_ = client.Close()
	}
}

func boolPtr(b bool) *bool { return &b }
