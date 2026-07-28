package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/antosc/aip/internal/alerts"
	"github.com/antosc/aip/internal/analytics/ingestion"
	"github.com/antosc/aip/internal/config"
	"github.com/antosc/aip/internal/logger"
	"github.com/antosc/aip/internal/notification"
	"github.com/antosc/aip/internal/repository/generator"
	"github.com/antosc/aip/internal/repository/postgres"
	"github.com/antosc/aip/internal/repository/towercore"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.LogLevel)

	db, err := postgres.New(cfg.Database.DSN())
	if err != nil {
		log.Error("ligação à base de dados falhou", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	client := towercore.NewHTTPClient(cfg.TowerCoreURL)
	generatorClient := generator.NewHTTPClient(cfg.GeneratorAPIURL)
	featureRepo := postgres.NewFeatureRepository(db)
	eventRepo := postgres.NewEventRepository(db)
	incidentRepo := postgres.NewIncidentCauseRepository(db)
	towerRepo := postgres.NewTowerRepository(db)

	// Alertas de limiar para o Teams (bateria, combustível) -- só ativa se
	// TEAMS_WEBHOOK_URL estiver configurado; sem isso, alertEngine fica nil
	// e a avaliação de alertas é ignorada sem erro.
	var alertEngine *alerts.Engine
	if cfg.TeamsWebhookURL != "" {
		teamsNotifier := notification.NewTeamsNotifier(cfg.TeamsWebhookURL)
		alertEngine = alerts.NewEngine(alerts.DefaultRules(), teamsNotifier, log)
	} else {
		log.Warn("TEAMS_WEBHOOK_URL não definido -- alertas de limiar desligados")
	}

	service := ingestion.NewService(client, generatorClient, featureRepo, eventRepo, incidentRepo, towerRepo, alertEngine, log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	log.Info("AIP scheduler a arrancar", "towercore_url", cfg.TowerCoreURL)

	// Primeiro ciclo imediato, sem esperar 5 minutos.
	runCycle(ctx, service, log)

	for {
		select {
		case <-ctx.Done():
			log.Info("AIP scheduler a encerrar")
			return
		case <-ticker.C:
			runCycle(ctx, service, log)
		}
	}
}

func runCycle(parent context.Context, service ingestion.Service, log *slog.Logger) {
	ctx, cancel := context.WithTimeout(parent, 2*time.Minute)
	defer cancel()

	if err := service.Collect(ctx); err != nil {
		log.Warn("ciclo de ingestão falhou", "err", err)
		return
	}

	log.Info("ciclo de ingestão concluído")
}
