// Package routes wires the HTTP API, its public endpoints, and its middleware.
package routes

import (
	"net/http"

	"towercore/internal/api/handlers"
	"towercore/internal/api/middleware"
	"towercore/internal/infrastructure/config"
	"towercore/internal/infrastructure/logger"
	"towercore/internal/observability"
)

type Handlers struct {
	NetworkLinks *handlers.NetworkLinksHandler
	ZabbixLinks *handlers.ZabbixLinksHandler
	Tower *handlers.TowerHandler
	TowerOperator *handlers.TowerOperatorHandler
	Event *handlers.EventHandler
	Metric *handlers.MetricHandler
	SNMPCollect *handlers.SNMPCollectHandler
	Audit *handlers.AuditHandler
	Auth *handlers.AuthHandler
	User *handlers.UserHandler
	Ticket *handlers.TicketHandler
	ComapReading *handlers.ComapReadingHandler
	DiscoveredDevice *handlers.DiscoveredDeviceHandler
	Region *handlers.RegionHandler
	Operator *handlers.OperatorHandler
	SLA *handlers.SLAHandler
	RadioKPI *handlers.RadioKPIHandler
	Backhaul *handlers.BackhaulInterfaceHandler
	SiteEnvironment *handlers.SiteEnvironmentHandler
}

// NewRouter builds the complete API surface. Health, metrics and login are
// public; operational endpoints require an authenticated user or service.
func NewRouter(cfg config.Config, log *logger.Logger, metrics *observability.Metrics, h Handlers) http.Handler {
	public := http.NewServeMux()
	health := handlers.NewHealthHandler(cfg)
	public.Handle("GET /healthz", health)
	public.Handle("GET /readyz", health)
	public.Handle("GET /metrics", metrics.Handler())
	public.Handle("POST /api/v1/auth", h.Auth)

	api := http.NewServeMux()
	api.HandleFunc("GET /api/v1/links", h.NetworkLinks.ListLinks)
	api.HandleFunc("GET /api/v1/links/{link_id}/events", h.NetworkLinks.GetLinkEvents)
	api.HandleFunc("GET /api/v1/links/{link_id}", h.NetworkLinks.GetLink)
	api.HandleFunc("POST /api/v1/links/sync/zabbix", h.ZabbixLinks.Sync)
	api.HandleFunc("GET /api/v1/links/zabbix/items", h.ZabbixLinks.Inspect)
	api.Handle("/api/v1/towers", h.Tower)
	api.Handle("/api/v1/towers/", h.Tower)
	api.Handle("POST /api/v1/towers/{id}/operators", h.TowerOperator)
	api.Handle("DELETE /api/v1/towers/{id}/operators/{operator_id}", h.TowerOperator)
	api.Handle("/api/v1/events", h.Event)
	api.Handle("/api/v1/events/", h.Event)
	api.Handle("/api/v1/metrics", h.Metric)
	api.Handle("/api/v1/metrics/", h.Metric)
	api.Handle("/api/v1/snmp/collect", h.SNMPCollect)
	api.Handle("/api/v1/audit", h.Audit)
	api.Handle("/api/v1/audit/", h.Audit)
	api.Handle("/api/v1/users", h.User)
	api.Handle("/api/v1/users/", h.User)
	api.Handle("/api/v1/tickets", h.Ticket)
	api.Handle("/api/v1/tickets/", h.Ticket)
	api.Handle("/api/v1/comap/readings", h.ComapReading)
	api.Handle("/api/v1/comap/readings/", h.ComapReading)
	api.Handle("/api/v1/discovered-devices", h.DiscoveredDevice)
	api.Handle("/api/v1/discovered-devices/", h.DiscoveredDevice)
	api.Handle("/api/v1/regions", h.Region)
	api.Handle("GET /api/v1/regions/{id}", http.HandlerFunc(h.Region.ServeByID))
	api.Handle("/api/v1/operators", h.Operator)
	api.Handle("GET /api/v1/operators/{id}", http.HandlerFunc(h.Operator.ServeByID))
	api.Handle("PUT /api/v1/operators/{id}", http.HandlerFunc(h.Operator.ServeByID))
	api.Handle("DELETE /api/v1/operators/{id}", http.HandlerFunc(h.Operator.ServeByID))
	api.Handle("/api/v1/sla", h.SLA)
	api.Handle("GET /api/v1/sla/region/{id}", http.HandlerFunc(h.SLA.ServeRegion))
	api.Handle("/api/v1/radio-kpi", h.RadioKPI)
	api.Handle("GET /api/v1/backhaul/tower/{id}/status", http.HandlerFunc(h.Backhaul.GetTowerBackhaulStatus))
	api.Handle("/api/v1/backhaul", h.Backhaul)
	api.Handle("GET /api/v1/site-environment/site/{id}/status", http.HandlerFunc(h.SiteEnvironment.GetSiteEnvironmentStatus))
	api.Handle("/api/v1/site-environment", h.SiteEnvironment)

	if cfg.Env == "prod" {
		public.Handle("/api/v1/", middleware.Chain(api,
			middleware.Auth(cfg.Auth.APIKeyHash, cfg.Auth.BearerToken, cfg.Auth.UserTokenSecret),
			middleware.RateLimitPerMinute(cfg.RateLimit.WritePerMinute),
		))
	} else {
		// In non-production environments we skip authentication to ease
		// local development and testing. Do NOT enable this in production.
		public.Handle("/api/v1/", middleware.Chain(api,
			middleware.RateLimitPerMinute(cfg.RateLimit.WritePerMinute),
		))
	}
	return middleware.Chain(public, middleware.RequestID(), middleware.AccessLog(log), middleware.CORS([]string{"http://localhost:3000", "http://localhost:8000","http://localhost:8080","http://172.21.1.106:8080","http://172.21.1.106:8000"}))
}
