package main

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"towercore/internal/infrastructure/logger"

	"go.uber.org/zap"

	"towercore/internal/adapters/eltek"
	"towercore/internal/adapters/enetek"
	"towercore/internal/adapters/huawei"
	"towercore/internal/adapters/nagios"
	"towercore/internal/adapters/neteco"
	"towercore/internal/adapters/snmp"
	"towercore/internal/adapters/vertiv"
	"towercore/internal/core/services"
	"towercore/internal/infrastructure/config"
	"towercore/internal/infrastructure/database"
	"towercore/internal/infrastructure/security"
	"towercore/internal/scheduler"
)

// cmd/scheduler é um processo leve, sem servidor HTTP, que corre local
// (na rede da Anglobal) e só faz polling dos equipamentos (SNMP, Nagios,
// ComAp/Modbus, NetEco) escrevendo diretamente na base de dados remota
// (RDS/EC2). Não aplica migrations (isso é responsabilidade do cmd/api,
// que já corre na EC2) e não expõe nenhuma rota HTTP.
func main() {
	cfg := config.Load()

	// Criar logger estruturado com Zap
	zapLog, err := zap.NewProduction()
	if err != nil {
		panic("failed to create zap logger: " + err.Error())
	}
	defer zapLog.Sync()

	// Adapter logger used by components that expect internal/logger.Logger
	stdLog := logger.New("")

	// Database — aponta para o Postgres remoto (EC2/RDS) via .env
	db, err := database.Open(cfg.DB)
	if err != nil {
		zapLog.Fatal("failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Security
	secretBox, err := security.NewSecretBox(cfg.SNMP.SecretKey)
	if err != nil {
		zapLog.Fatal("failed to initialize secret box", zap.Error(err))
	}

	// Repositories — só os necessários para os schedulers
	towerRepo := database.NewTowerRepository(db, secretBox)
	auditRepo := database.NewAuditRepository(db)
	eventRepo := database.NewEventRepository(db)
	metricRepo := database.NewMetricRepository(db)
	ticketRepo := database.NewTicketRepository(db)
	towerEndpointRepo := database.NewTowerEndpointRepository(db, secretBox)
	comapReadingRepo := database.NewComapReadingRepository(db)
	discoveredDeviceRepo := database.NewDiscoveredDeviceRepository(db, secretBox)

	// Core services (sem cache — não há handlers HTTP a beneficiar disso aqui)
	towerSvc := services.NewTowerService(towerRepo, auditRepo)
	eventSvc := services.NewEventService(eventRepo)
	metricSvc := services.NewMetricService(metricRepo)
	ticketSvc := services.NewTicketService(ticketRepo, auditRepo)

	// SNMP profiles
	snmpProfiles := map[string]snmp.Profile{
		"eltek":  eltek.Profile(),
		"huawei": huawei.Profile(),
		"enetek": enetek.Profile(),
		"vertiv": vertiv.Profile(),
	}

	snmpIngestSvc := services.NewSNMPIngestService(
		metricSvc,
		eventSvc,
		snmpProfiles,
		towerSvc,
		ticketSvc,
	)

	snmpScheduler := scheduler.NewSNMPScheduler(
		towerRepo,
		snmpIngestSvc,
		snmp.NewGoSNMPCollector(
			time.Duration(cfg.SNMP.TimeoutSeconds)*time.Second,
			cfg.SNMP.Retries,
		),
		snmpProfiles,
		zapLog,
		time.Duration(cfg.Scheduler.IntervalSeconds)*time.Second,
		cfg.Scheduler.BatchSize,
	)

	// Nagios
	nagiosClient := nagios.NewClient(
		cfg.Nagios.BaseURL,
		cfg.Nagios.Username,
		cfg.Nagios.Password,
		time.Duration(cfg.Nagios.TimeoutSeconds)*time.Second,
	)

	nagiosIngestSvc := services.NewNagiosIngestService(
		eventSvc,
		towerSvc,
	)

	nagiosScheduler := scheduler.NewNagiosScheduler(
		towerRepo,
		nagiosIngestSvc,
		nagiosClient,
		stdLog,
		time.Duration(cfg.Nagios.PollIntervalSeconds)*time.Second,
		cfg.Scheduler.BatchSize,
	)

	// ComAp (Modbus)
	comapIngestSvc := services.NewComapIngestService(
		comapReadingRepo,
		eventSvc,
		ticketSvc,
		towerSvc,
	)

	comapScheduler := scheduler.NewComapScheduler(
		towerEndpointRepo,
		comapIngestSvc,
		stdLog,
		time.Duration(cfg.Comap.IntervalSeconds)*time.Second,
		cfg.Scheduler.BatchSize,
		time.Duration(cfg.Comap.TimeoutSeconds)*time.Second,
	)

	// Discovery (opcional — só se for suposto correr a partir do local também)
	discoverySvc := services.NewDiscoveryService(
		snmp.NewGoSNMPProber(
			time.Duration(cfg.Discovery.TimeoutMillis)*time.Millisecond,
			cfg.SNMP.Retries,
		),
		discoveredDeviceRepo,
		snmp.NewEnterpriseVendorResolver(),
		cfg.Discovery.Community,
		cfg.Discovery.Concurrency,
		stdLog,
	)

	discoveryScheduler := scheduler.NewDiscoveryScheduler(
		discoverySvc,
		cfg.Discovery.CIDR,
		stdLog,
		time.Duration(cfg.Discovery.IntervalSeconds)*time.Second,
	)

	// NetEco (Huawei) — client de sessão + ingest de energia/bateria
	netEcoClient := neteco.NewClient(
		cfg.NetEco.BaseURL,
		cfg.NetEco.Username,
		cfg.NetEco.Password,
		cfg.NetEco.TLSInsecureSkipVerify,
		time.Duration(cfg.NetEco.TimeoutSeconds)*time.Second,
	)

	netEcoIngestSvc := services.NewNetEcoIngestService(
		netEcoClient,
		towerRepo,
		metricSvc,
	)

	netEcoAlarmSvc := services.NewNetEcoAlarmService(
		eventSvc,
		towerRepo,
		ticketSvc,
	)

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	zapLog.Info(
		"cmd/scheduler starting — coletor local (sem servidor HTTP)",
	)

	var snmpCancel context.CancelFunc = func() {}

	if cfg.Scheduler.Enabled {
		snmpCtx, cancel := context.WithCancel(context.Background())
		snmpCancel = cancel

		go snmpScheduler.Start(snmpCtx)
	} else {
		zapLog.Info("snmp scheduler disabled by configuration")
	}

	var discoveryCancel context.CancelFunc = func() {}

	if cfg.Discovery.Enabled {
		discoveryCtx, cancel := context.WithCancel(context.Background())
		discoveryCancel = cancel

		go discoveryScheduler.Start(discoveryCtx)
	} else {
		zapLog.Info("discovery scheduler disabled by configuration")
	}

	var comapCancel context.CancelFunc = func() {}

	if cfg.Comap.Enabled {
		go comapScheduler.Start(ctx)

		zapLog.Info(
			"comap scheduler worker successfully started in background",
		)
	} else {
		zapLog.Info(
			"comap scheduler skipped from startup context",
		)
	}

	var netEcoCancel context.CancelFunc = func() {}

	if cfg.NetEco.Enabled {
		if err := netEcoClient.Login(); err != nil {
			zapLog.Error(
				"neteco: initial login failed",
				zap.Error(err),
			)
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

		zapLog.Info(
			"neteco scheduler worker successfully started in background",
		)

		go func() {
			err := neteco.StartTrapListener(
				uint16(cfg.NetEco.TrapPort),
				cfg.NetEco.TrapCommunity,
				func(trap neteco.TrapEvent) {
					netEcoAlarmSvc.HandleTrap(
						context.Background(),
						trap,
					)
				},
			)

			if err != nil {
				zapLog.Error(
					"neteco: trap listener failed",
					zap.Error(err),
				)
			}
		}()

		zapLog.Info(
			"neteco trap listener started",
			zap.Int("port", cfg.NetEco.TrapPort),
		)
	} else {
		zapLog.Info(
			"neteco scheduler disabled by configuration",
		)
	}

	nagiosCtx, nagiosCancel := context.WithCancel(context.Background())

	go nagiosScheduler.Start(nagiosCtx)

	// Wait shutdown
	<-ctx.Done()

	zapLog.Info("shutdown signal received")

	snmpCancel()
	discoveryCancel()
	comapCancel()
	netEcoCancel()
	nagiosCancel()

	zapLog.Info("cmd/scheduler stopped")
}
