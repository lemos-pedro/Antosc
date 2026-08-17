


package main

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"towercore/internal/adapters/eltek"
	"towercore/internal/adapters/enetek"
	"towercore/internal/adapters/huawei"
	"towercore/internal/adapters/nagios"
	"towercore/internal/adapters/neteco"
	"towercore/internal/adapters/snmp"
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
	"towercore/internal/adapters/vertiv"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.LogLevel)

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

	// SNMP scheduler
	snmpScheduler := scheduler.NewSNMPScheduler(
		towerRepo,
		snmpIngestSvc,
		snmp.NewGoSNMPCollector(
			time.Duration(cfg.SNMP.TimeoutSeconds)*time.Second,
			cfg.SNMP.Retries,
		),
		snmpProfiles,
		log,
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

	discoveredDeviceHandler := handlers.NewDiscoveredDeviceHandler(
		discoveredDeviceRepo,
		promotionSvc,
	)

	regionHandler := handlers.NewRegionHandler(regionSvc, slaSvc)
	operatorHandler := handlers.NewOperatorHandler(operatorSvc)
	slaHandler := handlers.NewSLAHandler(slaSvc)

	// Router
	router := routes.NewRouter(
		cfg,
		log,
		metrics,
		towerHandler,
		towerOperatorHandler,
		eventHandler,
		metricHandler,
		snmpCollectHandler,
		auditHandler,
		authHandler,
		userHandler,
		ticketHandler,
		discoveredDeviceHandler,
		regionHandler,
		operatorHandler,
		slaHandler,
		comapReadingHandler,
		radioKPIHandler,
	)

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

	nagiosCtx, nagiosCancel := context.WithCancel(context.Background())
	go nagiosScheduler.Start(nagiosCtx)

	// Wait shutdown
	<-ctx.Done()
	log.Info("shutdown signal received")

	snmpCancel()
	discoveryCancel()
	comapCancel()
	netEcoCancel()
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