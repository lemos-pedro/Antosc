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
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.LogLevel)

	db, err := database.Open(cfg.DB)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		log.Fatalf("failed to apply database migrations: %v", err)
	}

	metrics, err := observability.New(cfg.AppName, db)
	if err != nil {
		log.Fatalf("failed to initialize observability: %v", err)
	}

	secretBox, err := security.NewSecretBox(cfg.SNMP.SecretKey)
	if err != nil {
		log.Fatalf("failed to initialize secret box: %v", err)
	}

	towerRepo := database.NewTowerRepository(db, secretBox)
	auditRepo := database.NewAuditRepository(db)
	eventRepo := database.NewEventRepository(db)
	metricRepo := database.NewMetricRepository(db)
	userRepo := database.NewUserRepository(db)
	ticketRepo := database.NewTicketRepository(db)
	discoveredDeviceRepo := database.NewDiscoveredDeviceRepository(db, secretBox)
	towerCache := cache.NewTowerCache(
		time.Duration(cfg.Cache.TowerListTTLSeconds)*time.Second,
		time.Duration(cfg.Cache.TowerDetailTTLSeconds)*time.Second,
	)

	towerSvc := services.NewTowerServiceWithCache(towerRepo, towerCache, auditRepo)
	eventSvc := services.NewEventService(eventRepo)
	metricSvc := services.NewMetricService(metricRepo)
	auditSvc := services.NewAuditService(auditRepo)
	ticketSvc := services.NewTicketService(ticketRepo, auditRepo)
	authSvc := services.NewAuthService(
		userRepo,
		cfg.Auth.UserTokenSecret,
		time.Duration(cfg.Auth.UserTokenTTLMin)*time.Minute,
	)
	if err := authSvc.EnsureBootstrapUser(
		context.Background(),
		cfg.Auth.BootstrapUsername,
		cfg.Auth.BootstrapPassword,
		cfg.Auth.BootstrapRole,
	); err != nil {
		log.Fatalf("failed to ensure bootstrap user: %v", err)
	}
	snmpProfiles := map[string]snmp.Profile{
		"eltek":  eltek.Profile(),
		"huawei": huawei.Profile(),
		"enetek": enetek.Profile(),
	}
	snmpIngestSvc := services.NewSNMPIngestService(metricSvc, eventSvc, snmpProfiles)

	// Nagios: ingest service e scheduler. towerSvc já implementa
	// UpdateStatus(ctx, towerID, status), por isso serve diretamente
	// como TowerStatusUpdater sem adapter extra.
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

	towerHandler := handlers.NewTowerHandler(towerSvc)
	eventHandler := handlers.NewEventHandler(eventSvc)
	metricHandler := handlers.NewMetricHandler(metricSvc)
	snmpCollectHandler := handlers.NewSNMPCollectHandler(snmpIngestSvc)
	auditHandler := handlers.NewAuditHandler(auditSvc)
	authHandler := handlers.NewAuthHandler(authSvc)
	userHandler := handlers.NewUserHandler(userRepo)
	ticketHandler := handlers.NewTicketHandler(ticketSvc)
	promotionSvc := services.NewDiscoveredDevicePromotionService(discoveredDeviceRepo, towerSvc)
	discoveredDeviceHandler := handlers.NewDiscoveredDeviceHandler(discoveredDeviceRepo, promotionSvc)
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

	discoverySvc := services.NewDiscoveryService(
		snmp.NewGoSNMPProber(time.Duration(cfg.Discovery.TimeoutMillis)*time.Millisecond, cfg.SNMP.Retries),
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

	router := routes.NewRouter(cfg, log, metrics, towerHandler, eventHandler, metricHandler, snmpCollectHandler, auditHandler, authHandler, userHandler, ticketHandler, discoveredDeviceHandler)
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Infof("starting http server on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server failed: %v", err)
		}
	}()

	var schedCancel context.CancelFunc = func() {}
	if cfg.Scheduler.Enabled {
		schedCtx, cancel := context.WithCancel(context.Background())
		schedCancel = cancel
		go snmpScheduler.Start(schedCtx)
	} else {
		log.Info("snmp scheduler disabled by configuration")
	}

	var discoveryCancel context.CancelFunc = func() {}
	if cfg.Discovery.Enabled {
		discCtx, cancel := context.WithCancel(context.Background())
		discoveryCancel = cancel
		go discoveryScheduler.Start(discCtx)
	} else {
		log.Info("discovery scheduler disabled by configuration")
	}

	// Nagios scheduler: sem flag de enable, corre sempre que a app arranca.
	nagiosCtx, nagiosCancel := context.WithCancel(context.Background())
	go nagiosScheduler.Start(nagiosCtx)

	<-ctx.Done()
	log.Info("shutdown signal received")
	schedCancel()
	discoveryCancel()
	nagiosCancel()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("failed to shutdown http server: %v", err)
	}

	log.Info("http server stopped")
}