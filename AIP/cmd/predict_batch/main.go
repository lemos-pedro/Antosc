// cmd/predict_batch corre previsões para todas as torres da cache local e
// grava em ai_predictions. Sem este job, /powerbi/predictions e a tool
// risk_ranking do assistente ficam vazios — ninguém faz POST manual por
// ~200 sites.
//
// Uso:
//
//	PREDICT_MODELS=health_score,anomaly,forecast \
//	PREDICT_WINDOW_DAYS=7 \
//	go run ./cmd/predict_batch
//
// Agendar via cron (ex: de 6 em 6 horas) ou Task Scheduler.
//
// Comportamento:
//   - Lista torres em cache (tabela towers)
//   - Para cada torre × modelo, chama prediction.PredictAndStore
//   - ErrInsufficientHistory / ErrUnknownVendor → log e continua (não aborta)
//   - No fim imprime contagens: ok / skip / erro
package main

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/antosc/aip/internal/config"
	"github.com/antosc/aip/internal/logger"
	"github.com/antosc/aip/internal/prediction"
	"github.com/antosc/aip/internal/repository/postgres"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.LogLevel)

	models := parseModels(os.Getenv("PREDICT_MODELS"))
	windowDays := 7
	if v := os.Getenv("PREDICT_WINDOW_DAYS"); v != "" {
		if n, err := parsePositiveInt(v); err == nil {
			windowDays = n
		}
	}

	db, err := postgres.New(cfg.Database.DSN())
	if err != nil {
		log.Error("ligação à base de dados falhou", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	featureRepo := postgres.NewFeatureRepository(db)
	predictionRepo := postgres.NewPredictionRepository(db)
	towerRepo := postgres.NewTowerRepository(db)

	svc := prediction.NewService(cfg.PredictionServiceURL, featureRepo, predictionRepo, towerRepo)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	towers, err := towerRepo.ListAll(ctx)
	if err != nil {
		log.Error("erro ao listar torres", "err", err)
		os.Exit(1)
	}
	if len(towers) == 0 {
		log.Warn("cache de torres vazia — corre o scheduler de ingestão primeiro")
		os.Exit(0)
	}

	log.Info("predict_batch a iniciar",
		"towers", len(towers),
		"models", models,
		"window_days", windowDays,
	)

	var ok, skip, fail int
	for _, t := range towers {
		for _, model := range models {
			if ctx.Err() != nil {
				log.Error("timeout global do batch", "err", ctx.Err())
				os.Exit(1)
			}

			towerCtx, towerCancel := context.WithTimeout(ctx, 45*time.Second)
			_, err := svc.PredictAndStore(towerCtx, t.TowerID, model, windowDays)
			towerCancel()

			if err != nil {
				if errors.Is(err, prediction.ErrInsufficientHistory) ||
					errors.Is(err, prediction.ErrUnknownVendor) {
					skip++
					log.Info("previsão omitida", "tower_id", t.TowerID, "model", model, "reason", err.Error())
					continue
				}
				fail++
				log.Warn("previsão falhou", "tower_id", t.TowerID, "model", model, "err", err)
				continue
			}
			ok++
			log.Info("previsão gravada", "tower_id", t.TowerID, "model", model)
		}
	}

	log.Info("predict_batch concluído",
		"ok", ok,
		"skip", skip,
		"fail", fail,
		"towers", len(towers),
	)
	if fail > 0 && ok == 0 {
		os.Exit(1)
	}
}

func parseModels(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{"health_score"}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{"health_score"}
	}
	return out
}

func parsePositiveInt(s string) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, errors.New("not a positive int")
		}
		n = n*10 + int(c-'0')
	}
	if n <= 0 {
		return 0, errors.New("not positive")
	}
	return n, nil
}
