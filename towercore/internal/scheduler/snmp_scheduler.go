package scheduler

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"go.uber.org/zap"

	"towercore/internal/adapters/comap"
	"towercore/internal/adapters/snmp"
	"towercore/internal/core/interfaces"
	"towercore/internal/core/services"
)

type SNMPScheduler struct {
	towersRepo    interfaces.TowerRepository
	ingestService *services.SNMPIngestService
	collector     snmp.Collector
	profiles      map[string]snmp.Profile
	log           *zap.Logger
	interval      time.Duration
	batchSize     int
}

func NewSNMPScheduler(
	towersRepo interfaces.TowerRepository,
	ingestService *services.SNMPIngestService,
	collector snmp.Collector,
	profiles map[string]snmp.Profile,
	log *zap.Logger,
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
		log: log.With(
			zap.String("component", "snmp_scheduler"),
		),
		interval:  interval,
		batchSize: batchSize,
	}
}

func (s *SNMPScheduler) Start(ctx context.Context) {
	s.log.Info(
		"snmp scheduler started",
		zap.String("interval", s.interval.String()),
		zap.Int("batch_size", s.batchSize),
	)

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
		start := time.Now()

		towers, total, err := s.towersRepo.List(
			ctx,
			interfaces.TowerFilter{
				Limit:  s.batchSize,
				Offset: offset,
			},
		)

		if err != nil {
			s.log.Error(
				"snmp scheduler list towers failed",
				zap.Error(err),
				zap.Int("offset", offset),
			)
			return
		}

		// Métrica: tempo de listagem
		listDur := time.Since(start)

		s.log.Debug(
			"towers list completed",
			zap.Int("count", len(towers)),
			zap.Int("total", total),
			zap.Duration("duration", listDur),
		)

		for _, tw := range towers {
			vendor := strings.ToLower(strings.TrimSpace(tw.Vendor))

			// Verificar latência de rede antes da coleta
			networkLatencyStart := time.Now()
			latencyErr := s.checkNetworkLatency(ctx, vendor, tw.SNMPTarget)
			networkLatencyDur := time.Since(networkLatencyStart)

			s.log.Debug(
				"network latency check completed",
				zap.String("tower_id", tw.ID),
				zap.String("vendor", vendor),
				zap.String("target", tw.SNMPTarget),
				zap.Duration("latency", networkLatencyDur),
				zap.Error(latencyErr),
			)

			// Torres ComAp utilizam Modbus em vez de SNMP.
			//
			// Não bloquear ComAp pelo SNMPEnabled, porque o equipamento
			// pode não utilizar SNMP.
			if vendor != "comap" && !tw.SNMPEnabled {
				continue
			}

			var samples = make([]domain.Metric, 0)
			var collectionErr error

			colStart := time.Now()

			switch vendor {
			case "comap":
				// Usar collector Modbus para equipamentos ComAp.
				comapCollector, err := comap.NewCollector(
					tw.SNMPTarget,
					5*time.Second,
				)

				if err != nil {
					s.log.Error(
						"comap scheduler failed to create collector",
						zap.String("tower_id", tw.ID),
						zap.String("target", tw.SNMPTarget),
						zap.Error(err),
					)

					continue
				}

				samples, collectionErr = comapCollector.Collect(
					ctx,
					tw,
					nil,
				)

				// Fechar imediatamente o collector para não acumular
				// conexões a cada ciclo do scheduler.
				if closeErr := comapCollector.Close(); closeErr != nil {
					s.log.Warn(
						"comap scheduler failed to close collector",
						zap.String("tower_id", tw.ID),
						zap.Error(closeErr),
					)
				}

			default:
				// Usar collector SNMP existente.
				profile, ok := s.profiles[vendor]

				if !ok {
					s.log.Warn(
						"snmp scheduler skipped tower",
						zap.String("tower_id", tw.ID),
						zap.String("vendor", tw.Vendor),
					)
					continue
				}

				samples, collectionErr = s.collector.Collect(
					ctx,
					tw,
					profile,
				)
			}

			colDur := time.Since(colStart)

			if collectionErr != nil {
				s.log.Error(
					"snmp scheduler collect failed",
					zap.String("tower_id", tw.ID),
					zap.String("vendor", vendor),
					zap.Error(collectionErr),
					zap.Duration("collection_duration", colDur),
				)

				// Marcar como unreachable
				if markErr := s.ingestService.MarkUnreachableWithError(
					ctx,
					tw.ID,
					collectionErr.Error(),
				); markErr != nil {
					s.log.Error(
						"snmp scheduler mark unreachable failed",
						zap.String("tower_id", tw.ID),
						zap.Error(markErr),
					)
				}

				continue
			}

			ingStart := time.Now()

			err = s.ingestService.Ingest(
				ctx,
				services.SNMPSnapshot{
					TowerID:     tw.ID,
					Vendor:      vendor,
					CollectedAt: time.Now().UTC(),
					Samples:     samples,
				},
			)

			ingDur := time.Since(ingStart)

			if err != nil {
				s.log.Error(
					"snmp scheduler ingest failed",
					zap.String("tower_id", tw.ID),
					zap.String("vendor", vendor),
					zap.Error(err),
					zap.Int("metrics_collected", len(samples)),
					zap.Duration("ingest_duration", ingDur),
				)

				if markErr := s.ingestService.MarkIngestConfigError(
					ctx,
					tw.ID,
					err.Error(),
				); markErr != nil {
					s.log.Error(
						"snmp scheduler mark ingest error failed",
						zap.String("tower_id", tw.ID),
						zap.Error(markErr),
					)
				}

				continue
			}

			s.log.Info(
				"snmp scheduler collected tower",
				zap.String("tower_id", tw.ID),
				zap.String("vendor", vendor),
				zap.Int("metrics", len(samples)),
				zap.Duration("collection_duration", colDur),
				zap.Duration("ingest_duration", ingDur),
			)
		}

		offset += len(towers)

		if len(towers) == 0 || offset >= total {
			return
		}
	}
}

// checkNetworkLatency faz uma verificação leve de latência/disponibilidade
// de rede antes da coleta, adaptando o protocolo/porta ao vendor.
//
// Nota: para UDP, DialContext não faz handshake — só confirma que o
// socket local abriu (resolução/rota ok), não que há um agente a
// responder do outro lado. Para uma verificação real de disponibilidade
// SNMP, considerar um Get leve (ex. sysUpTime.0) via s.collector.
func (s *SNMPScheduler) checkNetworkLatency(ctx context.Context, vendor, target string) error {
	dialer := net.Dialer{
		Timeout: 2 * time.Second,
	}

	var network, port string

	switch vendor {
	case "comap":
		// ComAp fala Modbus TCP, normalmente na porta 502.
		network, port = "tcp", "502"
	default:
		// SNMP normalmente corre sobre UDP na porta 161.
		network, port = "udp", "161"
	}

	conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(target, port))
	if err != nil {
		return fmt.Errorf("network check failed: %w", err)
	}
	conn.Close()

	return nil
}