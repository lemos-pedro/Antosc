package routes

import (
	"net/http"

	"github.com/antosc/aip/internal/api/handlers"
)

// Register liga todas as rotas do AIP a um *http.ServeMux (stdlib, Go 1.22+
// com r.PathValue, tal como já usado no towercore).
func Register(mux *http.ServeMux, predictionHandler *handlers.PredictionHandler) {
	mux.HandleFunc("GET /health", handlers.HealthHandler)

	mux.HandleFunc("GET /api/v1/predictions/{tower_id}", predictionHandler.GetLatest)
	mux.HandleFunc("POST /api/v1/predictions/{tower_id}", predictionHandler.Trigger)
}
