package handlers

import (
	"net/http"
	"strconv"

	"github.com/antosc/aip/internal/analytics"
	"github.com/antosc/aip/internal/repository/postgres"
)

type RiskHandler struct {
	predictions postgres.PredictionRepository
}

func NewRiskHandler(predictions postgres.PredictionRepository) *RiskHandler {
	return &RiskHandler{predictions: predictions}
}

// Portfolio — GET /api/v1/risk/portfolio?horizon_days=30&sims=5000
func (h *RiskHandler) Portfolio(w http.ResponseWriter, r *http.Request) {
	horizon, _ := strconv.Atoi(r.URL.Query().Get("horizon_days"))
	if horizon <= 0 {
		horizon = 30
	}
	sims, _ := strconv.Atoi(r.URL.Query().Get("sims"))
	if sims <= 0 {
		sims = 5000
	}
	if sims > 50000 {
		sims = 50000
	}

	preds, err := h.predictions.ListLatestPerTower(r.Context(), 5000)
	if err != nil {
		http.Error(w, "erro ao carregar previsões", http.StatusInternalServerError)
		return
	}
	if len(preds) == 0 {
		http.Error(w, "sem previsões — corre predict_batch primeiro", http.StatusNotFound)
		return
	}
	risk := analytics.MonteCarloPortfolio(preds, horizon, sims, 0)
	writeJSON(w, http.StatusOK, risk)
}
