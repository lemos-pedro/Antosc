package scheduler

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"towercore/internal/adapters/snmp"
	"towercore/internal/core/domain"
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

	// concurrency é o número máximo de torres processadas em paralelo por
	// ciclo. Antes desta mudança o loop era 100% sequencial: uma torre
	// inalcançável (timeout SNMP) bloqueava todas as seguintes, o que
	// media ciclos de horas em vez de minutos com ~194 torres e dezenas
	// delas sem rota de rede (ver runbook/GETIC). Valor default escolhido
	// como ponto de partida conservador — ajustar conforme CPU/rede
	// disponíveis, mas nunca tão alto que sature a rede local ou o EC2.
	concurrency int
}

// defaultConcurrency é o número de workers em paralelo quando não
// configurado explicitamente via NewSNMPScheduler.
const defaultConcurrency = 15

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
		interval:    interval,
		batchSize:   batchSize,
		concurrency: defaultConcurrency,
	}
}

// WithConcurrency permite ajustar o número de workers em paralelo. Chamar
// antes de Start. Valores <= 0 são ignorados (mantém o default).
func (s *SNMPScheduler) WithConcurrency(n int) *SNMPScheduler {
	if n > 0 {
		s.concurrency = n
	}
	return s
}

func (s *SNMPScheduler) Start(ctx context.Context) {
	s.log.Info(
		"snmp scheduler started",
		zap.String("interval", s.interval.String()),
		zap.Int("batch_size", s.batchSize),
		zap.Int("concurrency", s.concurrency),
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
	cycleStart := time.Now()

	var (
		collectedCount   int64
		unreachableCount int64
		skippedCount     int64
		errorCount       int64
		mu               sync.Mutex // protege os contadores acima
	)

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

		listDur := time.Since(start)

		s.log.Debug(
			"towers list completed",
			zap.Int("count", len(towers)),
			zap.Int("total", total),
			zap.Duration("duration", listDur),
		)

		// --- Worker pool: processa este batch de torres em paralelo ---
		//
		// Antes: for sequencial — uma torre lenta/inalcançável bloqueava
		// todas as seguintes, explicando ciclos de horas com dezenas de
		// torres sem rota de rede.
		//
		// Agora: até s.concurrency torres em voo ao mesmo tempo. O
		// semáforo (buffered channel) limita quantas goroutines correm
		// simultaneamente sem descontrolar o número total de goroutines
		// criadas.
		sem := make(chan struct{}, s.concurrency)
		var wg sync.WaitGroup

		for _, tw := range towers {
			tw := tw // captura por valor para a goroutine (evita partilha da variável do loop)

			wg.Add(1)
			sem <- struct{}{}

			go func() {
				defer wg.Done()
				defer func() { <-sem }()

				result := s.collectTower(ctx, tw)

				mu.Lock()
				switch result {
				case towerResultCollected:
					collectedCount++
				case towerResultUnreachable:
					unreachableCount++
				case towerResultSkipped:
					skippedCount++
				case towerResultError:
					errorCount++
				}
				mu.Unlock()
			}()
		}

		wg.Wait()

		offset += len(towers)

		if len(towers) == 0 || offset >= total {
			break
		}
	}

	s.log.Info(
		"snmp scheduler cycle completed",
		zap.Duration("cycle_total_duration", time.Since(cycleStart)),
		zap.Int64("collected", collectedCount),
		zap.Int64("unreachable", unreachableCount),
		zap.Int64("skipped", skippedCount),
		zap.Int64("errors", errorCount),
		zap.Int("concurrency", s.concurrency),
	)
}

type towerCollectResult int

const (
	towerResultCollected towerCollectResult = iota
	towerResultUnreachable
	towerResultSkipped
	towerResultError
)

// collectTower processa uma única torre: verificação de latência, coleta
// SNMP/Modbus e ingest. Isolado do loop principal para poder correr dentro
// de uma goroutine do worker pool sem partilhar estado mutável além dos
// contadores (protegidos por mutex no chamador).
func (s *SNMPScheduler) collectTower(ctx context.Context, tw domain.Tower) towerCollectResult {
	vendor := strings.ToLower(strings.TrimSpace(tw.Vendor))

	// Verificar latência de rede antes da coleta.
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

	// CORREÇÃO: antes o resultado de checkNetworkLatency era só logado e
	// nunca usado — a torre seguia sempre para collector.Collect(), que
	// repetia o mesmo timeout de rede (dial + Get). Agora, se a rede já
	// deu erro aqui, marcamos unreachable de imediato e poupamos o
	// timeout duplicado (tipicamente 5-10s por torre morta, multiplicado
	// por dezenas de torres sem rota).
	if latencyErr != nil {
		s.log.Warn(
			"snmp scheduler tower unreachable at network check, skipping collect",
			zap.String("tower_id", tw.ID),
			zap.String("vendor", vendor),
			zap.Error(latencyErr),
		)

		if markErr := s.ingestService.MarkUnreachableWithError(
			ctx,
			tw.ID,
			fmt.Sprintf("network check failed: %v", latencyErr),
		); markErr != nil {
			s.log.Error(
				"snmp scheduler mark unreachable failed",
				zap.String("tower_id", tw.ID),
				zap.Error(markErr),
			)
		}

		return towerResultUnreachable
	}

	// Torres ComAp utilizam Modbus em vez de SNMP.
	//
	// Não bloquear ComAp pelo SNMPEnabled, porque o equipamento
	// pode não utilizar SNMP.
	if vendor != "comap" && !tw.SNMPEnabled {
		return towerResultSkipped
	}

	var samples map[string]float64
	var collectionErr error

	colStart := time.Now()

	switch vendor {
	case "comap":
		s.log.Debug("comap vendor collection skipped (not implemented in scheduler)", zap.String("tower_id", tw.ID))
		return towerResultSkipped
	default:
		profile, ok := s.profiles[vendor]

		if !ok {
			s.log.Warn(
				"snmp scheduler skipped tower",
				zap.String("tower_id", tw.ID),
				zap.String("vendor", tw.Vendor),
			)
			return towerResultSkipped
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

		return towerResultUnreachable
	}

	ingStart := time.Now()

	err := s.ingestService.Ingest(
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

		return towerResultError
	}

	s.log.Info(
		"snmp scheduler collected tower",
		zap.String("tower_id", tw.ID),
		zap.String("vendor", vendor),
		zap.Int("metrics", len(samples)),
		zap.Duration("collection_duration", colDur),
		zap.Duration("ingest_duration", ingDur),
	)

	return towerResultCollected
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