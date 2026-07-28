package routes

import (
	"net/http"

	"github.com/antosc/aip/internal/api/handlers"
)

// Register liga todas as rotas do AIP a um *http.ServeMux (stdlib, Go 1.22+
// com r.PathValue, tal como já usado no towercore).
func Register(
	mux *http.ServeMux,
	predictionHandler *handlers.PredictionHandler,
	normHandler *handlers.ConsumptionNormHandler,
	incidentHandler *handlers.IncidentCauseHandler,
	deviationHandler *handlers.ConsumptionDeviationHandler,
	assistantHandler *handlers.AssistantHandler,
	exportHandler *handlers.ExportHandler,
) {
	mux.HandleFunc("GET /health", handlers.HealthHandler)

	mux.HandleFunc("GET /api/v1/predictions/{tower_id}", predictionHandler.GetLatest)
	mux.HandleFunc("POST /api/v1/predictions/{tower_id}", predictionHandler.Trigger)

	// Normas de consumo — geridas pelo Controller.
	mux.HandleFunc("POST /api/v1/norms", normHandler.Create)
	mux.HandleFunc("PUT /api/v1/norms/{id}", normHandler.Update)
	mux.HandleFunc("GET /api/v1/norms/{tower_id}", normHandler.ListByTower)
	mux.HandleFunc("DELETE /api/v1/norms/{id}", normHandler.Deactivate)

	// Causas de incidentes — ML + confirmação do O&M.
	mux.HandleFunc("POST /api/v1/incidents/{id}/confirm", incidentHandler.Confirm)
	mux.HandleFunc("GET /api/v1/incidents/pending", incidentHandler.PendingConfirmation)
	mux.HandleFunc("GET /api/v1/incidents", incidentHandler.ByPeriod)
	mux.HandleFunc("GET /api/v1/incidents/tower/{tower_id}", incidentHandler.ByTower)

	// Desvios de consumo — real vs norma do Controller. Base do relatório do Financeiro.
	mux.HandleFunc("GET /api/v1/consumption/deviations/{tower_id}", deviationHandler.ByTower)
	mux.HandleFunc("GET /api/v1/consumption/deviations/{tower_id}/compare", deviationHandler.Compare)

	// Assistente (Ollama) — Q&A livre por perfil, via function calling sobre as rotas acima.
	mux.HandleFunc("POST /api/v1/assistant/ask", assistantHandler.Ask)

	// Exportação para Excel sob pedido.
	mux.HandleFunc("GET /api/v1/reports/export/incidents", exportHandler.IncidentsExport)
	mux.HandleFunc("GET /api/v1/reports/export/consumption/{tower_id}", exportHandler.ConsumptionExport)
}
