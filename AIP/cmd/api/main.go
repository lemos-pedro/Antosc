package main

import (
	"net/http"
	"os"

	"github.com/antosc/aip/internal/api/handlers"
	"github.com/antosc/aip/internal/api/routes"
	"github.com/antosc/aip/internal/config"
	"github.com/antosc/aip/internal/logger"
	"github.com/antosc/aip/internal/prediction"
	"github.com/antosc/aip/internal/repository/postgres"
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

	predictionService := prediction.NewService(cfg.PredictionServiceURL, featureRepo, predictionRepo)
	predictionHandler := handlers.NewPredictionHandler(predictionRepo, predictionService, log)

	mux := http.NewServeMux()
	routes.Register(mux, predictionHandler)

	log.Info("AIP API a arrancar", "port", cfg.APIPort)

	if err := http.ListenAndServe(":"+cfg.APIPort, mux); err != nil {
		log.Error("servidor HTTP terminou com erro", "err", err)
		os.Exit(1)
	}
}
