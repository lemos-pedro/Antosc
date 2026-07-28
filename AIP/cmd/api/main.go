package main

import (
	"net/http"
	"os"

	"github.com/antosc/aip/internal/api/handlers"
	"github.com/antosc/aip/internal/api/routes"
	"github.com/antosc/aip/internal/assistant"
	"github.com/antosc/aip/internal/config"
	"github.com/antosc/aip/internal/logger"
	"github.com/antosc/aip/internal/prediction"
	"github.com/antosc/aip/internal/repository/postgres"
	"github.com/antosc/aip/internal/services/consumption"
	"github.com/antosc/aip/internal/tools"
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

	featureRepo := postgres.NewFeatureRepository(db)
	predictionRepo := postgres.NewPredictionRepository(db)
	towerRepo := postgres.NewTowerRepository(db)
	normRepo := postgres.NewConsumptionNormRepository(db)
	incidentRepo := postgres.NewIncidentCauseRepository(db)
	auditRepo := postgres.NewAssistantAuditRepository(db)

	predictionService := prediction.NewService(cfg.PredictionServiceURL, featureRepo, predictionRepo, towerRepo)
	predictionHandler := handlers.NewPredictionHandler(predictionRepo, predictionService, log)
	normHandler := handlers.NewConsumptionNormHandler(normRepo)
	incidentHandler := handlers.NewIncidentCauseHandler(incidentRepo)

	deviationService := consumption.NewService(normRepo, featureRepo)
	deviationHandler := handlers.NewConsumptionDeviationHandler(deviationService)

	toolRegistry := tools.NewRegistry(cfg.AIPBaseURL)
	ollamaClient := assistant.NewOllamaClient(cfg.OllamaURL, cfg.OllamaModel)
	auditLogger := handlers.NewAuditAdapter(auditRepo)
	assistantService := assistant.NewService(ollamaClient, toolRegistry, auditLogger, log)
	assistantHandler := handlers.NewAssistantHandler(assistantService, log)

	exportHandler := handlers.NewExportHandler(incidentRepo, deviationService)

	mux := http.NewServeMux()
	routes.Register(mux, predictionHandler, normHandler, incidentHandler, deviationHandler, assistantHandler, exportHandler)

	log.Info("AIP API a arrancar", "port", cfg.APIPort)

	if err := http.ListenAndServe(":"+cfg.APIPort, mux); err != nil {
		log.Error("servidor HTTP terminou com erro", "err", err)
		os.Exit(1)
	}
}
