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

type Handlers struct {
	NetworkLinks     *handlers.NetworkLinksHandler
	ZabbixLinks      *handlers.ZabbixLinksHandler
	Tower            *handlers.TowerHandler
	TowerOperator    *handlers.TowerOperatorHandler
	Event            *handlers.EventHandler
	Metric           *handlers.MetricHandler
	SNMPCollect      *handlers.SNMPCollectHandler
	Audit            *handlers.AuditHandler
	Auth             *handlers.AuthHandler
	User             *handlers.UserHandler
	Ticket           *handlers.TicketHandler
	ComapReading     *handlers.ComapReadingHandler
	DiscoveredDevice *handlers.DiscoveredDeviceHandler
	Region           *handlers.RegionHandler
	Operator         *handlers.OperatorHandler
	SLA              *handlers.SLAHandler
	RadioKPI         *handlers.RadioKPIHandler
	Backhaul         *handlers.BackhaulInterfaceHandler
	SiteEnvironment  *handlers.SiteEnvironmentHandler
}

func NewRouter(
	cfg config.Config,
	log *logger.Logger,
	metrics *observability.Metrics,
	h Handlers,
) http.Handler {

	mux := http.NewServeMux()

	writeChain := func(handler http.Handler) http.Handler {
		return middleware.Chain(handler, middleware.Auth(cfg.Auth.APIKeyHash, cfg.Auth.BearerToken, cfg.Auth.UserTokenSecret), middleware.RateLimitPerMinute(cfg.RateLimit.WritePerMinute))
	}

	healthHandler := handlers.NewHealthHandler(cfg)

	// ---------------------------------------------------------
	// HEALTH / METRICS
	// ---------------------------------------------------------

	mux.Handle("GET /api/v1/health", metrics.Instrument("/api/v1/health", healthHandler))
	mux.Handle("GET /health", metrics.Instrument("/health", healthHandler))
	mux.Handle("GET /metrics", metrics.Instrument("/metrics", metrics.Handler()))

	// ---------------------------------------------------------
	// AUTH
	// ---------------------------------------------------------

	mux.Handle("POST /api/v1/auth/login", metrics.Instrument("/api/v1/auth/login", h.Auth))

	// ---------------------------------------------------------
	// USERS
	// ---------------------------------------------------------

	mux.Handle("GET /api/v1/users", metrics.Instrument("/api/v1/users", writeChain(middleware.RequireRole("admin")(h.User))))
	mux.Handle("POST /api/v1/users", metrics.Instrument("/api/v1/users", writeChain(middleware.RequireRole("admin")(h.User))))

	// ---------------------------------------------------------
	// TOWERS
	// ---------------------------------------------------------

	mux.Handle("GET /api/v1/towers", metrics.Instrument("/api/v1/towers", h.Tower))

	mux.Handle("GET /api/v1/towers/{id}", metrics.Instrument("/api/v1/towers/{id}", h.Tower))

	mux.Handle("POST /api/v1/towers", metrics.Instrument("/api/v1/towers", writeChain(h.Tower)))

	mux.Handle("PATCH /api/v1/towers/{id}/snmp", metrics.Instrument("/api/v1/towers/{id}/snmp", writeChain(h.Tower)))

	mux.Handle("POST /api/v1/towers/{id}/operators", metrics.Instrument("/api/v1/towers/{id}/operators", h.TowerOperator))

	mux.Handle("DELETE /api/v1/towers/{id}/operators/{operator_id}", metrics.Instrument("/api/v1/towers/{id}/operators/{operator_id}", h.TowerOperator))

	// ---------------------------------------------------------
	// EVENTS
	// ---------------------------------------------------------

	mux.Handle("GET /api/v1/events", metrics.Instrument("/api/v1/events", h.Event))

	mux.Handle("POST /api/v1/events", metrics.Instrument("/api/v1/events", writeChain(h.Event)))

	mux.Handle("GET /api/v1/towers/{id}/events", metrics.Instrument("/api/v1/towers/{id}/events", http.HandlerFunc(h.Event.ListByTower)))

	// ---------------------------------------------------------
	// METRICS
	// ---------------------------------------------------------

	mux.Handle("GET /api/v1/metrics", metrics.Instrument("/api/v1/metrics", h.Metric))
	mux.Handle("POST /api/v1/metrics", metrics.Instrument("/api/v1/metrics", writeChain(h.Metric)))

	// ---------------------------------------------------------
	// SNMP
	// ---------------------------------------------------------

	mux.Handle("POST /api/v1/collect/snmp", metrics.Instrument("/api/v1/collect/snmp", writeChain(h.SNMPCollect)))
	// Compatibilidade com a rota nova do GitHub
	mux.Handle("POST /api/v1/snmp/collect", metrics.Instrument("/api/v1/snmp/collect", writeChain(h.SNMPCollect)))

	// ---------------------------------------------------------
	// TICKETS
	// ---------------------------------------------------------

	mux.Handle("GET /api/v1/tickets", metrics.Instrument("/api/v1/tickets", h.Ticket))
	mux.Handle("POST /api/v1/tickets/{ticket_id}/ack", metrics.Instrument("/api/v1/tickets/{ticket_id}/ack", h.Ticket))
	mux.Handle("POST /api/v1/tickets/{ticket_id}/close", metrics.Instrument("/api/v1/tickets/{ticket_id}/close", h.Ticket))

	// ---------------------------------------------------------
	// AUDIT
	// ---------------------------------------------------------

	mux.Handle("GET /api/v1/audit-logs", metrics.Instrument("/api/v1/audit-logs", writeChain(h.Audit)))
	mux.Handle("GET /api/v1/audit-logs/{id}", metrics.Instrument("/api/v1/audit-logs/{id}", writeChain(h.Audit)))
	mux.Handle("GET /api/v1/audit-logs/export.csv", metrics.Instrument("/api/v1/audit-logs/export.csv", writeChain(h.Audit)))

	// ---------------------------------------------------------
	// DISCOVERED DEVICES
	// ---------------------------------------------------------

	mux.Handle("GET /api/v1/discovered-devices", metrics.Instrument("/api/v1/discovered-devices", http.HandlerFunc(h.DiscoveredDevice.List)))
	mux.Handle("POST /api/v1/discovered-devices/{id}/promote", metrics.Instrument("/api/v1/discovered-devices/{id}/promote", writeChain(http.HandlerFunc(h.DiscoveredDevice.Promote))))
	mux.Handle("POST /api/v1/discovered-devices/{id}/ignore", metrics.Instrument("/api/v1/discovered-devices/{id}/ignore", writeChain(http.HandlerFunc(h.DiscoveredDevice.Ignore))))

	// ---------------------------------------------------------
	// OPERATORS
	// ---------------------------------------------------------

	mux.Handle("GET /api/v1/operators", metrics.Instrument("/api/v1/operators", h.Operator))
	mux.Handle("POST /api/v1/operators", metrics.Instrument("/api/v1/operators", writeChain(h.Operator)))
	mux.Handle("GET /api/v1/operators/{id}", metrics.Instrument("/api/v1/operators/{id}", http.HandlerFunc(h.Operator.ServeByID)))
	mux.Handle("PUT /api/v1/operators/{id}", metrics.Instrument("/api/v1/operators/{id}", writeChain(http.HandlerFunc(h.Operator.ServeByID))))
	mux.Handle("DELETE /api/v1/operators/{id}", metrics.Instrument("/api/v1/operators/{id}", writeChain(http.HandlerFunc(h.Operator.ServeByID))))

	// ---------------------------------------------------------
	// REGIONS
	// ---------------------------------------------------------

	mux.Handle("GET /api/v1/regions", metrics.Instrument("/api/v1/regions", h.Region))
	mux.Handle("POST /api/v1/regions", metrics.Instrument("/api/v1/regions", writeChain(h.Region)))
	mux.Handle("GET /api/v1/regions/{id}", metrics.Instrument("/api/v1/regions/{id}", http.HandlerFunc(h.Region.ServeByID)))

	// ---------------------------------------------------------
	// SLA
	// ---------------------------------------------------------

	mux.Handle("GET /api/v1/sla/global", metrics.Instrument("/api/v1/sla/global", h.SLA))
	mux.Handle("GET /api/v1/sla/region/{id}", metrics.Instrument("/api/v1/sla/region/{id}", http.HandlerFunc(h.SLA.ServeRegion)))

	// ---------------------------------------------------------
	// RADIO KPI
	// ---------------------------------------------------------

	mux.Handle("GET /api/v1/radio-kpi", metrics.Instrument("/api/v1/radio-kpi", h.RadioKPI))

	// ---------------------------------------------------------
	// BACKHAUL
	// ---------------------------------------------------------

	mux.Handle("GET /api/v1/backhaul", metrics.Instrument("/api/v1/backhaul", h.Backhaul))

	mux.Handle("GET /api/v1/backhaul/tower/{id}/status", metrics.Instrument("/api/v1/backhaul/tower/{id}/status", http.HandlerFunc(h.Backhaul.GetTowerBackhaulStatus)))

	// ---------------------------------------------------------
	// SITE ENVIRONMENT
	// ---------------------------------------------------------

	mux.Handle("GET /api/v1/site-environment", metrics.Instrument("/api/v1/site-environment", h.SiteEnvironment))
	mux.Handle("GET /api/v1/site-environment/site/{id}/status", metrics.Instrument("/api/v1/site-environment/site/{id}/status", http.HandlerFunc(h.SiteEnvironment.GetSiteEnvironmentStatus)))

	// ---------------------------------------------------------
	// COMAP / GENERATOR
	// ---------------------------------------------------------

	mux.Handle("GET /api/v1/towers/{tower_id}/energy/generator", metrics.Instrument("/api/v1/towers/{tower_id}/energy/generator", http.HandlerFunc(h.ComapReading.GetByTowerID)))

	// ---------------------------------------------------------
	// NETWORK LINKS / ZABBIX
	// ---------------------------------------------------------

	mux.Handle(
		"GET /api/v1/links",
		metrics.Instrument("/api/v1/links", http.HandlerFunc(h.NetworkLinks.ListLinks)),
	)
	mux.Handle(
		"GET /api/v1/links/{link_id}",
		metrics.Instrument("/api/v1/links/{link_id}", http.HandlerFunc(h.NetworkLinks.GetLink)),
	)
	mux.Handle(
		"GET /api/v1/links/{link_id}/events",
		metrics.Instrument("/api/v1/links/{link_id}/events", http.HandlerFunc(h.NetworkLinks.GetLinkEvents)),
	)
	mux.Handle(
		"POST /api/v1/links/sync/zabbix",
		metrics.Instrument("/api/v1/links/sync/zabbix", writeChain(http.HandlerFunc(h.ZabbixLinks.Sync))),
	)
	mux.Handle(
		"GET /api/v1/links/zabbix/items",
		metrics.Instrument("/api/v1/links/zabbix/items", http.HandlerFunc(h.ZabbixLinks.Inspect)),
	)

	// ---------------------------------------------------------
	// 404
	// ---------------------------------------------------------

	mux.Handle("/", metrics.Instrument("not_found", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Errorf("route not found: method=%s path=%s", r.Method, r.URL.Path)
		apierror.Write(w, http.StatusNotFound, "not_found", "route not found")
	})))

	// ---------------------------------------------------------
	// FINAL MIDDLEWARE PIPELINE
	// ---------------------------------------------------------

	return middleware.Chain(
		mux,
		middleware.CORS([]string{
			"http://172.21.1.106:8080",
		}),
		middleware.RequestID(),
		middleware.AccessLog(log),
	)
}
