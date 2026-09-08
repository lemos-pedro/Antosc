package scheduler

import (
	"context"
	"time"

	"towercore/internal/core/services"
	"towercore/internal/infrastructure/logger"
)

// ZabbixScheduler sincroniza links de rede a partir do Zabbix periodicamente.
//
// CORRIGIDO: a versão anterior chamava
// client.GetHostsByNameFilter([]string{""}) — uma string de busca vazia,
// que nunca isolava os hosts certos (Benguela/Huambo) — e o corpo de
// runOnce nunca chegou a ser terminado (`_ = items`, sem persistir nada).
// Resultado: o scheduler nem sequer estava registado no main.go, e os
// snapshots em link_metric_snapshots só existiam porque alguém chamou
// manualmente o endpoint POST /api/v1/zabbix/links/sync (dois clusters
// isolados de timestamps, sem cadência).
//
// Esta versão delega toda a lógica de host filtering, agrupamento por
// ifIndex e persistência (links + snapshots) a ZabbixLinkSyncService, que
// já trata isso corretamente (ver zabbix_link_sync.go). O scheduler só
// decide "quando" correr, não "como".
type ZabbixScheduler struct {
	svc      *services.ZabbixLinkSyncService
	searches []string
	interval time.Duration
	enabled  bool
	logger   *logger.Logger
}

func NewZabbixScheduler(
	svc *services.ZabbixLinkSyncService,
	searches []string,
	interval time.Duration,
	enabled bool,
	logger *logger.Logger,
) *ZabbixScheduler {
	return &ZabbixScheduler{
		svc:      svc,
		searches: searches,
		interval: interval,
		enabled:  enabled,
		logger:   logger,
	}
}

func (s *ZabbixScheduler) Start(ctx context.Context) {
	if !s.enabled {
		s.logger.Info("zabbix scheduler disabled by configuration")
		return
	}

	if len(s.searches) == 0 {
		s.logger.Errorf("zabbix scheduler: ZABBIX_HOST_SEARCH está vazio — a não arrancar para evitar importar todos os hosts do Zabbix (ex.: ZABBIX_HOST_SEARCH=Benguela,Huambo)")
		return
	}

	s.logger.Infof("zabbix scheduler started (host_search=%v, interval=%s)", s.searches, s.interval)

	// Primeira sincronização imediata — não esperar pelo primeiro tick
	// para ter dados, mesmo padrão dos outros schedulers do projeto.
	s.runOnce(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("zabbix scheduler stopping")
			return
		case <-ticker.C:
			s.runOnce(ctx)
		}
	}
}

func (s *ZabbixScheduler) runOnce(ctx context.Context) {
	result, err := s.svc.Sync(ctx, s.searches)
	if err != nil {
		s.logger.Errorf("zabbix scheduler: sync failed: %v", err)
		return
	}

	s.logger.Infof(
		"zabbix scheduler: sync done — hosts=%d links_created=%d links_updated=%d snapshots=%d errors=%d",
		result.Hosts, result.LinksCreated, result.LinksUpdated, result.Snapshots, len(result.Errors),
	)

	for _, e := range result.Errors {
		s.logger.Errorf("zabbix scheduler: %s", e)
	}
}