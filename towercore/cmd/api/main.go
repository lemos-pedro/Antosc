package main

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"

	"towercore/internal/adapters/eltek"
	"towercore/internal/adapters/enetek"
	"towercore/internal/adapters/hizima"
	"towercore/internal/adapters/huawei"
	"towercore/internal/adapters/nagios"
	"towercore/internal/adapters/neteco"
	"towercore/internal/adapters/snmp"
	"towercore/internal/adapters/vertiv"
	"towercore/internal/adapters/zabbix"
	"towercore/internal/api/handlers"
	"towercore/internal/api/routes"
	"towercore/internal/core/services"
	"towercore/internal/infrastructure/cache"
	"towercore/internal/infrastructure/config"
	"towercore/internal/infrastructure/database"
	"towercore/internal/infrastructure/logger"
	"towercore/internal/infrastructure/security"
	"towercore/internal/observability"
	"towercore/internal/scheduler"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.LogLevel)
	zapLog, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to initialize structured logger: %v", err)
	}
	defer zapLog.Sync()

	// Database
	db, err := database.Open(cfg.DB)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		log.Fatalf("failed to apply database migrations: %v", err)
	}

	// Observability
	metrics, err := observability.New(cfg.AppName, db)
	if err != nil {
		log.Fatalf("failed to initialize observability: %v", err)
	}

	// Security
	secretBox, err := security.NewSecretBox(cfg.SNMP.SecretKey)
	if err != nil {
		log.Fatalf("failed to initialize secret box: %v", err)
	}

	// Repositories
	towerRepo := database.NewTowerRepository(db, secretBox)
	auditRepo := database.NewAuditRepository(db)
	eventRepo := database.NewEventRepository(db)
	metricRepo := database.NewMetricRepository(db)
	userRepo := database.NewUserRepository(db)
	ticketRepo := database.NewTicketRepository(db)
	discoveredDeviceRepo := database.NewDiscoveredDeviceRepository(db, secretBox)
	towerEndpointRepo := database.NewTowerEndpointRepository(db, secretBox)
	comapReadingRepo := database.NewComapReadingRepository(db)
	backhaulRepo := database.NewBackhaulInterfaceRepository(db)
	siteEnvironmentRepo := database.NewSiteEnvironmentRepository(db)
	networkLinkRepo := database.NewNetworkLinkRepository(db)
	networkLinkEventRepo := database.NewNetworkLinkEventRepository(db)

	regionRepo := database.NewRegionRepository(db)
	operatorRepo := database.NewOperatorRepository(db)
	slaRepo := database.NewSLARepository(db)

	// Cache
	towerCache := cache.NewTowerCache(
		time.Duration(cfg.Cache.TowerListTTLSeconds)*time.Second,
		time.Duration(cfg.Cache.TowerDetailTTLSeconds)*time.Second,
	)

	// Core services
	towerSvc := services.NewTowerServiceWithCache(towerRepo, towerCache, auditRepo)
	eventSvc := services.NewEventService(eventRepo)
	metricSvc := services.NewMetricService(metricRepo)
	auditSvc := services.NewAuditService(auditRepo)
	ticketSvc := services.NewTicketService(ticketRepo, auditRepo)
	availabilitySvc := services.NewAvailabilityService(eventRepo, towerRepo, metricRepo)

	// Radio KPI service
	radioKPIRepo := database.NewRadioKPIRepository(db)
	radioKPISvc := services.NewRadioKPIService(radioKPIRepo)
	backhaulSvc := services.NewBackhaulInterfaceService(backhaulRepo)
	siteEnvironmentSvc := services.NewSiteEnvironmentService(siteEnvironmentRepo)

	// Zabbix — client + serviço de sincronização de links de rede
	// (network_links + link_metric_snapshots). O scheduler correspondente
	// é registado mais abaixo, junto aos outros pollers.
	zabbixClient := zabbix.NewClient(cfg.Zabbix.BaseURL, cfg.Zabbix.Username, cfg.Zabbix.Password, cfg.Zabbix.APIToken, time.Duration(cfg.Zabbix.TimeoutSeconds)*time.Second)
	zabbixLinkSvc := services.NewZabbixLinkSyncService(zabbixClient, networkLinkRepo, log)

	// NetEco (Huawei) — bateria e energia DC via API interna do NetEco +
	// alarmes via SNMP trap. Enabled=false por default (NETECO_ENABLED) —
	// depende de credenciais de sessão válidas (login/authenticate.action),
	// não NBI oficial.
	netEcoClient := neteco.NewClient(
		cfg.NetEco.BaseURL,
		cfg.NetEco.Username,
		cfg.NetEco.Password,
		cfg.NetEco.TLSInsecureSkipVerify,
		time.Duration(cfg.NetEco.TimeoutSeconds)*time.Second,
	)
	netEcoIngestSvc := services.NewNetEcoIngestService(netEcoClient, towerRepo, metricSvc)
	netEcoAlarmSvc := services.NewNetEcoAlarmService(eventSvc, towerRepo, ticketSvc)

	authSvc := services.NewAuthService(
		userRepo,
		cfg.Auth.UserTokenSecret,
		time.Duration(cfg.Auth.UserTokenTTLMin)*time.Minute,
	)

	regionSvc := services.NewRegionService(regionRepo)
	operatorSvc := services.NewOperatorService(operatorRepo)
	slaSvc := services.NewSLAService(slaRepo)

	// Bootstrap user
	if err := authSvc.EnsureBootstrapUser(
		context.Background(),
		cfg.Auth.BootstrapUsername,
		cfg.Auth.BootstrapPassword,
		cfg.Auth.BootstrapRole,
	); err != nil {
		log.Fatalf("failed to ensure bootstrap user: %v", err)
	}

	// Hizima (ZMACS) — API read-only para estado de locks, eventos de
	// acesso e work orders. Sem endpoint de controlo remoto (a app mobile
	// do técnico é quem tranca/destranca via BLE local) — este client
	// serve apenas consulta/auditoria, nunca comando. Enabled=false por
	// default (HIZIMA_ENABLED) — reservado para uso futuro de scheduler;
	// os handlers HTTP funcionam independentemente disso, pois chamam a
	// API ao vivo a cada pedido.
	hizimaClient := hizima.NewClient(hizima.Config{
		Host:     cfg.Hizima.Host,
		ClientID: cfg.Hizima.ClientID,
		Security: cfg.Hizima.Security,
		Username: cfg.Hizima.Username,
		Password: cfg.Hizima.Password,
	}, nil)

	// resolveStationNo: mapeamento tower_id (UUID interno) -> sno (StationNo
	// Hizima). Config estática via HIZIMA_STATION_MAP ("uuid1:sno1,uuid2:sno2")
	// — sem tabela nova em base de dados. Migra-se para coluna/tabela própria
	// só se crescer demasiado para caber em env var.
	hizimaStationMap := parseStationMap(cfg.Hizima.StationMap)
	resolveStationNo := func(towerID string) (string, bool) {
		v, ok := hizimaStationMap[towerID]
		return v, ok
	}

	// SNMP profiles
	snmpProfiles := map[string]snmp.Profile{
		"eltek":  eltek.Profile(),
		"huawei": huawei.Profile(),
		"enetek": enetek.Profile(),
		"vertiv": vertiv.Profile(),
	}

	// SNMP services
	snmpIngestSvc := services.NewSNMPIngestService(metricSvc, eventSvc, snmpProfiles, towerSvc, ticketSvc)

	// Nagios
	nagiosClient := nagios.NewClient(
		cfg.Nagios.BaseURL,
		cfg.Nagios.Username,
		cfg.Nagios.Password,
		time.Duration(cfg.Nagios.TimeoutSeconds)*time.Second,
	)

	nagiosIngestSvc := services.NewNagiosIngestService(eventSvc, towerSvc)

	nagiosScheduler := scheduler.NewNagiosScheduler(
		towerRepo,
		nagiosIngestSvc,
		nagiosClient,
		log,
		time.Duration(cfg.Nagios.PollIntervalSeconds)*time.Second,
		cfg.Scheduler.BatchSize,
	)

	// Promotion
	promotionSvc := services.NewDiscoveredDevicePromotionService(
		discoveredDeviceRepo,
		towerSvc,
	)

	// Discovery
	discoverySvc := services.NewDiscoveryService(
		snmp.NewGoSNMPProber(
			time.Duration(cfg.Discovery.TimeoutMillis)*time.Millisecond,
			cfg.SNMP.Retries,
		),
		discoveredDeviceRepo,
		snmp.NewEnterpriseVendorResolver(),
		cfg.Discovery.Community,
		cfg.Discovery.Concurrency,
		log,
	)

	discoveryScheduler := scheduler.NewDiscoveryScheduler(
		discoverySvc,
		cfg.Discovery.CIDR,
		log,
		time.Duration(cfg.Discovery.IntervalSeconds)*time.Second,
	)

	// SNMP collector (shared) and scheduler
	collector := snmp.NewGoSNMPCollector(
		time.Duration(cfg.SNMP.TimeoutSeconds)*time.Second,
		cfg.SNMP.Retries,
	)

	snmpScheduler := scheduler.NewSNMPScheduler(
		towerRepo,
		snmpIngestSvc,
		collector,
		snmpProfiles,
		zapLog,
		time.Duration(cfg.Scheduler.IntervalSeconds)*time.Second,
		cfg.Scheduler.BatchSize,
	)

	// Schedulers for Radio KPI, Backhaul interfaces and Site Environment
	radioScheduler := scheduler.NewRadioScheduler(
		towerRepo,
		radioKPISvc,
		collector,
		snmpProfiles,
		zapLog,
		time.Duration(cfg.Scheduler.IntervalSeconds)*time.Second,
		cfg.Scheduler.BatchSize,
	)

	backhaulScheduler := scheduler.NewBackhaulScheduler(
		towerRepo,
		backhaulSvc,
		collector,
		snmpProfiles,
		zapLog,
		time.Duration(cfg.Scheduler.IntervalSeconds)*time.Second,
		cfg.Scheduler.BatchSize,
	)

	siteEnvironmentScheduler := scheduler.NewSiteEnvironmentScheduler(
		towerRepo,
		siteEnvironmentSvc,
		collector,
		snmpProfiles,
		zapLog,
		time.Duration(cfg.Scheduler.IntervalSeconds)*time.Second,
		cfg.Scheduler.BatchSize,
	)

	// ComAp (Modbus) — telemetria de energia do grupo gerador.
	// Enabled=false por default (COMAP_ENABLED) até validação de campo
	// confirmar fatores de escala de fuel_percent/battery_voltage.
	comapIngestSvc := services.NewComapIngestService(comapReadingRepo, eventSvc, ticketSvc, towerSvc)
	comapScheduler := scheduler.NewComapScheduler(
		towerEndpointRepo,
		comapIngestSvc,
		log,
		time.Duration(cfg.Comap.IntervalSeconds)*time.Second,
		cfg.Scheduler.BatchSize,
		time.Duration(cfg.Comap.TimeoutSeconds)*time.Second,
	)

	// Zabbix link scheduler — sincroniza network_links + link_metric_snapshots
	// a partir do Zabbix (host.get + item.get) periodicamente. Substitui a
	// dependência de chamadas manuais a POST /api/v1/zabbix/links/sync, que
	// era a única forma de os dados entrarem até agora (ver os dois
	// timestamps isolados em link_metric_snapshots: 15:06:33 e 15:10:25,
	// sem cadência regular). Enabled=false por default (ZABBIX_ENABLED) até
	// confirmares o ZABBIX_HOST_SEARCH certo (ex.: "Benguela,Huambo") — o
	// scheduler recusa-se a arrancar com host_search vazio para não
	// importar todos os hosts do Zabbix.
	zabbixScheduler := scheduler.NewZabbixScheduler(
		zabbixLinkSvc,
		splitConfigList(cfg.Zabbix.HostSearch),
		time.Duration(cfg.Zabbix.IntervalSeconds)*time.Second,
		cfg.Zabbix.Enabled,
		log,
	)

	// Handlers
	towerHandler := handlers.NewTowerHandler(towerSvc, availabilitySvc)
	towerOperatorHandler := handlers.NewTowerOperatorHandler(towerSvc)
	eventHandler := handlers.NewEventHandler(eventSvc)
	metricHandler := handlers.NewMetricHandler(metricSvc)
	snmpCollectHandler := handlers.NewSNMPCollectHandler(snmpIngestSvc)
	auditHandler := handlers.NewAuditHandler(auditSvc)
	authHandler := handlers.NewAuthHandler(authSvc)
	userHandler := handlers.NewUserHandler(userRepo)
	ticketHandler := handlers.NewTicketHandler(ticketSvc)
	comapReadingHandler := handlers.NewComapReadingHandler(comapReadingRepo)

	// Radio KPI handler
	radioKPIHandler := handlers.NewRadioKPIHandler(radioKPISvc)
	backhaulHandler := handlers.NewBackhaulInterfaceHandler(backhaulSvc)
	siteEnvironmentHandler := handlers.NewSiteEnvironmentHandler(siteEnvironmentSvc)
	networkLinksHandler := handlers.NewNetworkLinksHandler(networkLinkRepo, networkLinkEventRepo)
	zabbixLinksHandler := handlers.NewZabbixLinksHandler(zabbixLinkSvc, splitConfigList(cfg.Zabbix.HostSearch))

	discoveredDeviceHandler := handlers.NewDiscoveredDeviceHandler(
		discoveredDeviceRepo,
		promotionSvc,
	)

	regionHandler := handlers.NewRegionHandler(regionSvc, slaSvc)
	operatorHandler := handlers.NewOperatorHandler(operatorSvc)
	slaHandler := handlers.NewSLAHandler(slaSvc)

	// Hizima (ZMACS) handler — expõe lock-status/lock-events/work-orders
	// por torre ao dashboard. Read-only, sem persistência própria.
	lockHandler := handlers.NewLockHandler(hizimaClient, resolveStationNo)

	// Router
	router := routes.NewRouter(cfg, log, metrics, routes.Handlers{
		NetworkLinks:     networkLinksHandler,
		ZabbixLinks:      zabbixLinksHandler,
		Tower:            towerHandler,
		TowerOperator:    towerOperatorHandler,
		Event:            eventHandler,
		Metric:           metricHandler,
		SNMPCollect:      snmpCollectHandler,
		Audit:            auditHandler,
		Auth:             authHandler,
		User:             userHandler,
		Ticket:           ticketHandler,
		ComapReading:     comapReadingHandler,
		DiscoveredDevice: discoveredDeviceHandler,
		Region:           regionHandler,
		Operator:         operatorHandler,
		SLA:              slaHandler,
		RadioKPI:         radioKPIHandler,
		Backhaul:         backhaulHandler,
		SiteEnvironment:  siteEnvironmentHandler,
		Lock:             lockHandler,
	})

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	go func() {
		log.Infof("starting http server on :%s", cfg.Port)

		if err := server.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server failed: %v", err)
		}
	}()

	// Start schedulers
	var snmpCancel context.CancelFunc = func() {}
	if cfg.Scheduler.Enabled {
		snmpCtx, cancel := context.WithCancel(context.Background())
		snmpCancel = cancel
		go snmpScheduler.Start(snmpCtx)
		go radioScheduler.Start(snmpCtx)
		go backhaulScheduler.Start(snmpCtx)
		go siteEnvironmentScheduler.Start(snmpCtx)
	} else {
		log.Info("snmp scheduler disabled by configuration")
	}

	var discoveryCancel context.CancelFunc = func() {}
	if cfg.Discovery.Enabled {
		discoveryCtx, cancel := context.WithCancel(context.Background())
		discoveryCancel = cancel
		go discoveryScheduler.Start(discoveryCtx)
	} else {
		log.Info("discovery scheduler disabled by configuration")
	}

	var comapCancel context.CancelFunc = func() {}
	if cfg.Comap.Enabled {
		go comapScheduler.Start(ctx)
		log.Info("comap scheduler worker successfully started in background")
	} else {
		log.Info("comap scheduler skipped from startup context")
	}

	// NetEco — scheduler de energia/bateria + trap listener de alarmes,
	// controlados pelo mesmo NETECO_ENABLED.
	var netEcoCancel context.CancelFunc = func() {}
	if cfg.NetEco.Enabled {
		if err := netEcoClient.Login(); err != nil {
			log.Errorf("neteco: initial login failed: %v", err)
		}

		netEcoCtx, cancel := context.WithCancel(context.Background())
		netEcoCancel = cancel

		go func() {
			interval := time.Duration(cfg.NetEco.IntervalSeconds) * time.Second
			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			for {
				select {
				case <-netEcoCtx.Done():
					return
				case <-ticker.C:
					netEcoIngestSvc.PollEnergyStatus(netEcoCtx)
				}
			}
		}()
		log.Info("neteco scheduler worker successfully started in background")

		go func() {
			err := neteco.StartTrapListener(uint16(cfg.NetEco.TrapPort), cfg.NetEco.TrapCommunity, func(trap neteco.TrapEvent) {
				netEcoAlarmSvc.HandleTrap(context.Background(), trap)
			})
			if err != nil {
				log.Errorf("neteco: trap listener failed: %v", err)
			}
		}()
		log.Infof("neteco trap listener started on :%d", cfg.NetEco.TrapPort)
	} else {
		log.Info("neteco scheduler disabled by configuration")
	}

	// Zabbix link scheduler — só arranca se ZABBIX_ENABLED=true. O próprio
	// scheduler valida internamente se ZABBIX_HOST_SEARCH está preenchido
	// antes de correr a primeira sincronização.
	var zabbixCancel context.CancelFunc = func() {}
	if cfg.Zabbix.Enabled {
		zabbixCtx, cancel := context.WithCancel(context.Background())
		zabbixCancel = cancel
		go zabbixScheduler.Start(zabbixCtx)
		log.Info("zabbix link scheduler successfully started in background")
	} else {
		log.Info("zabbix link scheduler disabled by configuration")
	}

	nagiosCtx, nagiosCancel := context.WithCancel(context.Background())
	go nagiosScheduler.Start(nagiosCtx)

	// Wait shutdown
	<-ctx.Done()
	log.Info("shutdown signal received")

	snmpCancel()
	discoveryCancel()
	comapCancel()
	netEcoCancel()
	zabbixCancel()
	nagiosCancel()

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		cfg.ShutdownTimeout,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("failed to shutdown http server: %v", err)
	}

	log.Info("http server stopped")
}

func splitConfigList(raw string) []string {
	var out []string
	for _, value := range strings.Split(raw, ",") {
		if value = strings.TrimSpace(value); value != "" {
			out = append(out, value)
		}
	}
	return out
}

// parseStationMap lê o formato "tower_id1:sno1,tower_id2:sno2" de
// HIZIMA_STATION_MAP e devolve o mapa pronto a usar. Entradas malformadas
// são ignoradas — nunca fabricar uma correspondência incerta.
func parseStationMap(raw string) map[string]string {
	out := make(map[string]string)
	for _, pair := range strings.Split(raw, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) != 2 {
			continue
		}
		towerID := strings.TrimSpace(parts[0])
		sno := strings.TrimSpace(parts[1])
		if towerID == "" || sno == "" {
			continue
		}
		out[towerID] = sno
	}
	return out
}