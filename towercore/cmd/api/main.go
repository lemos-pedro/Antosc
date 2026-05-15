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
	"towercore/internal/adapters/snmp"
	"towercore/internal/api/handlers"
	"towercore/internal/api/routes"
	"towercore/internal/core/services"
	"towercore/internal/infrastructure/config"
	"towercore/internal/infrastructure/database"
	"towercore/internal/infrastructure/logger"
	"towercore/internal/infrastructure/security"
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

	secretBox, err := security.NewSecretBox(cfg.SNMP.SecretKey)
	if err != nil {
		log.Fatalf("failed to initialize secret box: %v", err)
	}

	towerRepo := database.NewTowerRepository(db, secretBox)
	auditRepo := database.NewAuditRepository(db)
	eventRepo := database.NewEventRepository(db)
	metricRepo := database.NewMetricRepository(db)
	userRepo := database.NewUserRepository(db)

	towerSvc := services.NewTowerService(towerRepo, auditRepo)
	eventSvc := services.NewEventService(eventRepo)
	metricSvc := services.NewMetricService(metricRepo)
	auditSvc := services.NewAuditService(auditRepo)
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

	towerHandler := handlers.NewTowerHandler(towerSvc)
	eventHandler := handlers.NewEventHandler(eventSvc)
	metricHandler := handlers.NewMetricHandler(metricSvc)
	snmpCollectHandler := handlers.NewSNMPCollectHandler(snmpIngestSvc)
	auditHandler := handlers.NewAuditHandler(auditSvc)
	authHandler := handlers.NewAuthHandler(authSvc)
	snmpScheduler := scheduler.NewSNMPScheduler(
		towerRepo,
		snmpIngestSvc,
		snmp.NewFallbackCollector(
			snmp.NewGoSNMPCollector(
				time.Duration(cfg.SNMP.TimeoutSeconds)*time.Second,
				cfg.SNMP.Retries,
			),
			snmp.NewSyntheticCollector(),
		),
		snmpProfiles,
		log,
		time.Duration(cfg.Scheduler.IntervalSeconds)*time.Second,
		cfg.Scheduler.BatchSize,
	)

	router := routes.NewRouter(cfg, log, towerHandler, eventHandler, metricHandler, snmpCollectHandler, auditHandler, authHandler)

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

	schedCtx, schedCancel := context.WithCancel(context.Background())
	go snmpScheduler.Start(schedCtx)

	<-ctx.Done()
	log.Info("shutdown signal received")
	schedCancel()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("failed to shutdown http server: %v", err)
	}

	log.Info("http server stopped")
}
