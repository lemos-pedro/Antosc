package routes

import (
	"net/http"

	"towercore/internal/api/handlers"
	"towercore/internal/api/middleware"
	"towercore/internal/infrastructure/config"
	"towercore/internal/infrastructure/logger"
	"towercore/internal/observability"
	"towercore/pkg/apierror"
)

func NewRouter(
	cfg config.Config,
	log *logger.Logger,
	metrics *observability.Metrics,
	towerHandler *handlers.TowerHandler,
	towerOperatorHandler *handlers.TowerOperatorHandler, 
	eventHandler *handlers.EventHandler,
	metricHandler *handlers.MetricHandler,
	snmpCollectHandler *handlers.SNMPCollectHandler,
	auditHandler *handlers.AuditHandler,
	authHandler *handlers.AuthHandler,
	userHandler *handlers.UserHandler,
	ticketHandler *handlers.TicketHandler,
	discoveredDeviceHandler *handlers.DiscoveredDeviceHandler,
	regionHandler *handlers.RegionHandler,
	operatorHandler *handlers.OperatorHandler,
	slaHandler *handlers.SLAHandler,
	comapReadingHandler *handlers.ComapReadingHandler,
	
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

	mux.Handle("GET /api/v1/health", metrics.Instrument("/api/v1/health", healthHandler))
	mux.Handle("GET /health", metrics.Instrument("/health", healthHandler))
	mux.Handle("GET /metrics", metrics.Instrument("/metrics", metrics.Handler()))

	mux.Handle("POST /api/v1/users", metrics.Instrument("/api/v1/users", writeChain(middleware.RequireRole("admin")(userHandler))))
	mux.Handle("POST /api/v1/auth/login", metrics.Instrument("/api/v1/auth/login", authHandler))

	mux.Handle("GET /api/v1/towers", metrics.Instrument("/api/v1/towers", towerHandler))
	mux.Handle("GET /api/v1/towers/{id}", metrics.Instrument("/api/v1/towers/{id}", towerHandler))
	mux.Handle("POST /api/v1/towers", metrics.Instrument("/api/v1/towers", writeChain(towerHandler))) 
	mux.Handle("PATCH /api/v1/towers/{id}/snmp", metrics.Instrument("/api/v1/towers/{id}/snmp", writeChain(towerHandler)))

	mux.Handle("GET /api/v1/events", metrics.Instrument("/api/v1/events", eventHandler))
	mux.Handle("POST /api/v1/events", metrics.Instrument("/api/v1/events", writeChain(eventHandler)))
	mux.Handle("GET /api/v1/towers/{id}/events", metrics.Instrument("/api/v1/towers/{id}/events", http.HandlerFunc(eventHandler.ListByTower)))

	mux.Handle("GET /api/v1/metrics", metrics.Instrument("/api/v1/metrics", metricHandler))
	mux.Handle("POST /api/v1/metrics", metrics.Instrument("/api/v1/metrics", writeChain(metricHandler)))

	mux.Handle("POST /api/v1/collect/snmp", metrics.Instrument("/api/v1/collect/snmp", writeChain(snmpCollectHandler)))

	mux.Handle("GET /api/v1/tickets", metrics.Instrument("/api/v1/tickets", ticketHandler))
	mux.Handle("POST /api/v1/tickets/{ticket_id}/ack", metrics.Instrument("/api/v1/tickets/{ticket_id}/ack",ticketHandler))
	mux.Handle("POST /api/v1/tickets/{ticket_id}/close", metrics.Instrument("/api/v1/tickets/{ticket_id}/close",ticketHandler))

	mux.Handle("GET /api/v1/audit-logs", metrics.Instrument("/api/v1/audit-logs", writeChain(auditHandler)))
	mux.Handle("GET /api/v1/audit-logs/{id}", metrics.Instrument("/api/v1/audit-logs/{id}", writeChain(auditHandler)))
	mux.Handle("GET /api/v1/audit-logs/export.csv", metrics.Instrument("/api/v1/audit-logs/export.csv", writeChain(auditHandler)))

	mux.Handle("GET /api/v1/discovered-devices", metrics.Instrument("/api/v1/discovered-devices", http.HandlerFunc(discoveredDeviceHandler.List)))
	mux.Handle("POST /api/v1/discovered-devices/{id}/promote", metrics.Instrument("/api/v1/discovered-devices/{id}/promote", writeChain(http.HandlerFunc(discoveredDeviceHandler.Promote))))
	mux.Handle("POST /api/v1/discovered-devices/{id}/ignore", metrics.Instrument("/api/v1/discovered-devices/{id}/ignore", writeChain(http.HandlerFunc(discoveredDeviceHandler.Ignore))))

	mux.Handle("GET /api/v1/operators", metrics.Instrument("/api/v1/operators", operatorHandler)) 
	mux.Handle("POST /api/v1/operators", metrics.Instrument("/api/v1/operators", operatorHandler)) 
	mux.Handle("POST /api/v1/towers/{id}/operators", metrics.Instrument("/api/v1/towers/{id}/operators", towerOperatorHandler)) 
	mux.Handle("DELETE /api/v1/towers/{id}/operators/{operator_id}", metrics.Instrument("/api/v1/towers/{id}/operators/{operator_id}", towerOperatorHandler))

	mux.Handle("GET /api/v1/regions", metrics.Instrument("/api/v1/regions", regionHandler))
	mux.Handle("POST /api/v1/regions", metrics.Instrument("/api/v1/regions", writeChain(regionHandler)))

	mux.Handle("GET /api/v1/towers/{tower_id}/energy/generator", metrics.Instrument("/api/v1/towers/{tower_id}/energy/generator", http.HandlerFunc(comapReadingHandler.GetByTowerID)))
	
	mux.Handle("GET /api/v1/sla/global",metrics.Instrument("/api/v1/sla/global", slaHandler))

	mux.Handle("GET /api/v1/sla/region/{id}",metrics.Instrument("/api/v1/sla/region/{id}", http.HandlerFunc(slaHandler.ServeRegion)))
	

	mux.Handle("/", metrics.Instrument("not_found", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Errorf("route not found: method=%s path=%s", r.Method, r.URL.Path)
		apierror.Write(w, http.StatusNotFound, "not_found", "route not found")
	})))

	//  PIPELINE FINAL (CORS primeiro na cadeia)
	return middleware.Chain(
		mux,
		middleware.CORS([]string{
			"http://localhost:8080",
			"http://localhost:8081",
			"https://antosc-livid.vercel.app",
		}),
		middleware.RequestID(),
		middleware.AccessLog(log),
	)
}