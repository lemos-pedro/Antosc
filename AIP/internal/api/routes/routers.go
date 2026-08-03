package routes

import (
	"net/http"

	"github.com/antosc/aip/internal/api/handlers"
	"github.com/antosc/aip/internal/api/middleware"
	"github.com/antosc/aip/internal/auth"
)

// Register aplica autenticação + RBAC.
//
// Matriz (admin tem acesso a tudo):
//
//	Normas (CRUD)     → controller
//	Confirmar causa   → om, diretor_tecnico
//	Trigger predict   → engenharia, diretor_tecnico
//	Assistente        → qualquer autenticado (role do token)
//	Leituras / BI     → JWT ou API key
func Register(
	mux *http.ServeMux,
	predictionHandler *handlers.PredictionHandler,
	normHandler *handlers.ConsumptionNormHandler,
	incidentHandler *handlers.IncidentCauseHandler,
	deviationHandler *handlers.ConsumptionDeviationHandler,
	assistantHandler *handlers.AssistantHandler,
	exportHandler *handlers.ExportHandler,
	powerBIHandler *handlers.PowerBIHandler,
	authHandler *handlers.AuthHandler,
	riskHandler *handlers.RiskHandler,
	embeddingsHandler *handlers.EmbeddingsHandler,
	tokens *auth.TokenService,
	apiKey string,
) {
	jwt := middleware.Authenticate(tokens)
	admin := func(h http.HandlerFunc) http.HandlerFunc {
		return jwt(middleware.RequireAdmin(h))
	}
	// jwt + roles (admin implícito)
	role := func(roles ...string) func(http.HandlerFunc) http.HandlerFunc {
		return func(h http.HandlerFunc) http.HandlerFunc {
			return jwt(middleware.RequireRoles(roles...)(h))
		}
	}
	read := middleware.AuthenticateOrAPIKey(tokens, apiKey)

	mux.HandleFunc("GET /health", handlers.HealthHandler)

	// Auth público
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)
	mux.HandleFunc("POST /api/v1/auth/bootstrap", authHandler.Bootstrap)
	mux.HandleFunc("POST /api/v1/auth/refresh", authHandler.Refresh)

	// Auth autenticado / admin
	mux.HandleFunc("GET /api/v1/auth/me", jwt(authHandler.Me))
	mux.HandleFunc("POST /api/v1/auth/logout", jwt(authHandler.Logout))
	mux.HandleFunc("POST /api/v1/auth/change-password", jwt(authHandler.ChangePassword))
	mux.HandleFunc("POST /api/v1/auth/users", admin(authHandler.CreateUser))
	mux.HandleFunc("GET /api/v1/auth/users", admin(authHandler.ListUsers))
	mux.HandleFunc("PATCH /api/v1/auth/users/{id}", admin(authHandler.UpdateUser))
	mux.HandleFunc("GET /api/v1/auth/audit", admin(authHandler.ListAuthAudit))

	// Predictions
	mux.HandleFunc("GET /api/v1/predictions/{tower_id}", read(predictionHandler.GetLatest))
	mux.HandleFunc("POST /api/v1/predictions/{tower_id}", role("engenharia", "diretor_tecnico")(predictionHandler.Trigger))

	// Normas — Controller
	mux.HandleFunc("POST /api/v1/norms", role("controller")(normHandler.Create))
	mux.HandleFunc("PUT /api/v1/norms/{id}", role("controller")(normHandler.Update))
	mux.HandleFunc("GET /api/v1/norms/{tower_id}", jwt(normHandler.ListByTower))
	mux.HandleFunc("DELETE /api/v1/norms/{id}", role("controller")(normHandler.Deactivate))

	// Incidentes — confirmar: O&M + Diretor Técnico
	mux.HandleFunc("POST /api/v1/incidents/{id}/confirm", role("om", "diretor_tecnico")(incidentHandler.Confirm))
	mux.HandleFunc("GET /api/v1/incidents/pending", read(incidentHandler.PendingConfirmation))
	mux.HandleFunc("GET /api/v1/incidents", read(incidentHandler.ByPeriod))
	mux.HandleFunc("GET /api/v1/incidents/tower/{tower_id}", read(incidentHandler.ByTower))

	// Consumo
	mux.HandleFunc("GET /api/v1/consumption/deviations/{tower_id}", read(deviationHandler.ByTower))
	mux.HandleFunc("GET /api/v1/consumption/deviations/{tower_id}/compare", read(deviationHandler.Compare))

	// Assistente — qualquer user autenticado
	mux.HandleFunc("POST /api/v1/assistant/ask", jwt(assistantHandler.Ask))

	// Exports + Power BI
	mux.HandleFunc("GET /api/v1/reports/export/incidents", read(exportHandler.IncidentsExport))
	mux.HandleFunc("GET /api/v1/reports/export/consumption/{tower_id}", read(exportHandler.ConsumptionExport))
	mux.HandleFunc("GET /api/v1/powerbi/catalog", read(powerBIHandler.Catalog))
	mux.HandleFunc("GET /api/v1/powerbi/incidents", read(powerBIHandler.Incidents))
	mux.HandleFunc("GET /api/v1/powerbi/kpis", read(powerBIHandler.KPIs))
	mux.HandleFunc("GET /api/v1/powerbi/consumption", read(powerBIHandler.Consumption))
	mux.HandleFunc("GET /api/v1/powerbi/predictions", read(powerBIHandler.Predictions))
	mux.HandleFunc("GET /api/v1/powerbi/towers", read(powerBIHandler.Towers))

	// Risk portfolio (Monte Carlo) + embeddings
	mux.HandleFunc("GET /api/v1/risk/portfolio", jwt(riskHandler.Portfolio))
	mux.HandleFunc("GET /api/v1/towers/{tower_id}/similar", read(embeddingsHandler.Similar))
}
