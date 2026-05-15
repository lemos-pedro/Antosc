package routes

import (
	"net/http"

	"towercore/internal/api/handlers"
	"towercore/internal/api/middleware"
	"towercore/internal/infrastructure/config"
	"towercore/internal/infrastructure/logger"
	"towercore/pkg/apierror"
)

func NewRouter(
	cfg config.Config,
	log *logger.Logger,
	towerHandler *handlers.TowerHandler,
	eventHandler *handlers.EventHandler,
	metricHandler *handlers.MetricHandler,
	snmpCollectHandler *handlers.SNMPCollectHandler,
	auditHandler *handlers.AuditHandler,
	authHandler *handlers.AuthHandler,
) http.Handler {
	mux := http.NewServeMux()
	writeChain := func(h http.Handler) http.Handler {
		return middleware.Chain(
			h,
			middleware.Auth(cfg.Auth.APIKeyHash, cfg.Auth.BearerToken, cfg.Auth.UserTokenSecret),
			middleware.RateLimitPerMinute(cfg.RateLimit.WritePerMinute),
		)
	}

	healthHandler := handlers.NewHealthHandler(cfg)

	mux.Handle("GET /api/v1/health", healthHandler)
	mux.Handle("GET /health", healthHandler)
	mux.Handle("POST /api/v1/auth/login", authHandler)
	mux.Handle("GET /api/v1/towers", towerHandler)
	mux.Handle("GET /api/v1/towers/{id}", towerHandler)
	mux.Handle("POST /api/v1/towers", writeChain(towerHandler))
	mux.Handle("PATCH /api/v1/towers/{id}/snmp", writeChain(towerHandler))
	mux.Handle("GET /api/v1/events", eventHandler)
	mux.Handle("POST /api/v1/events", writeChain(eventHandler))
	mux.Handle("GET /api/v1/metrics", metricHandler)
	mux.Handle("POST /api/v1/metrics", writeChain(metricHandler))
	mux.Handle("POST /api/v1/collect/snmp", writeChain(snmpCollectHandler))
	mux.Handle("GET /api/v1/audit-logs", writeChain(auditHandler))
	mux.Handle("GET /api/v1/audit-logs/{id}", writeChain(auditHandler))
	mux.Handle("GET /api/v1/audit-logs/export.csv", writeChain(auditHandler))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Errorf("route not found: method=%s path=%s", r.Method, r.URL.Path)
		apierror.Write(w, http.StatusNotFound, "not_found", "route not found")
	})

	return middleware.Chain(
		mux,
		middleware.RequestID(),
		middleware.AccessLog(log),
	)
}
