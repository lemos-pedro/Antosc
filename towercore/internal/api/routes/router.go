// Package routes contém o roteamento HTTP e middlewares para a API.
package routes

import (
	"net/http"

	"towercore/internal/api/handlers"
	"towercore/internal/core/services"
)

// Router define os caminhos HTTP e seus handlers correspondentes.
type Router struct {
	// Handlers
	towerHandler          *handlers.TowerHandler
	towerOperatorHandler  *handlers.TowerOperatorHandler
	eventHandler          *handlers.EventHandler
	metricHandler         *handlers.MetricHandler
	snmpCollectHandler    *handlers.SNMPCollectHandler
	auditHandler          *handlers.AuditHandler
	authHandler           *handlers.AuthHandler
	userHandler           *handlers.UserHandler
	ticketHandler         *handlers.TicketHandler
	comapReadingHandler   *handlers.ComapReadingHandler
	discoveredDeviceHandler *handlers.DiscoveredDeviceHandler
	regionHandler         *handlers.RegionHandler
	operatorHandler       *handlers.OperatorHandler
	slaHandler            *handlers.SLAHandler
	radioKPIHandler       *handlers.RadioKPIHandler
	backhaulInterfaceHandler *handlers.BackhaulInterfaceHandler
	siteEnvironmentHandler *handlers.SiteEnvironmentHandler // <-- NOVO HANDLER DE AMBIENTE

	// Middleware
	logger  *zap.Logger
	metrics *observability.Metrics
}

// NewRouter cria um novo router com os handlers e middlewares especificados.
func NewRouter(
	cfg *config.Config,
	logger *zap.Logger,
	metrics *observability.Metrics,
	towerHandler *handlers.TowerHandler,
	towerOperatorHandler *handlers.TowerOperatorHandler,
	eventHandler *handlers.EventHandler,
	metricHandler *handlers.MetricHandler,
	snmpCollectHandler *handlers.SNMPCollectHandler,
	auditHandler *handlers.AuditHandler,
	authHandler          *handlers.AuthHandler,
	userHandler          *handlers.UserHandler,
	ticketHandler        *handlers.TicketHandler,
	comapReadingHandler  *handlers.ComapReadingHandler,
	discoveredDeviceHandler *handlers.DiscoveredDeviceHandler,
	regionHandler        *handlers.RegionHandler,
	operatorHandler      *handlers.OperatorHandler,
	slaHandler           *handlers.SLAHandler,
	radioKPIHandler      *handlers.RadioKPIHandler,
	backhaulInterfaceHandler *handlers.BackhaulInterfaceHandler,
	siteEnvironmentHandler *handlers.SiteEnvironmentHandler, // <-- NOVO HANDLER DE AMBIENTE
) *Router {
	return &Router{
		// Handlers
		towerHandler:          towerHandler,
		towerOperatorHandler:  towerOperatorHandler,
		eventHandler:          eventHandler,
		metricHandler:         metricHandler,
		snmpCollectHandler:    snmpCollectHandler,
		auditHandler:          auditHandler,
		authHandler:           authHandler,
		userHandler:           userHandler,
		ticketHandler:         ticketHandler,
		comapReadingHandler:   comapreadingHandler,
		discoveredDeviceHandler: discoveredDeviceHandler,
		regionHandler:         regionHandler,
		operatorHandler:       operatorHandler,
		slaHandler:            slaHandler,
		radioKPIHandler:       radioKPIHandler,
		backhaulInterfaceHandler: backhaulInterfaceHandler,
		siteEnvironmentHandler: siteEnvironmentHandler, // <-- NOVO HANDLER DE AMBIENTE

		// Middleware
		logger:  logger,
		metrics: metrics,
	}
}

// ServeHTTP implementa a interface http.Handler.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Aplicar middlewares aqui (logging, métricas, etc.)
	// Então encaminhar para o handler apropriado baseado no path
	path := req.URL.Path

	switch {
	case path == "/api/towers" || HasPrefix(path, "/api/towers/"):
		r.towerHandler.ServeHTTP(w, req)
	case path == "/api/events" || HasPrefix(path, "/api/events/"):
		r.eventHandler.ServeHTTP(w, req)
	case path == "/api/metrics" || HasPrefix(path, "/api/metrics/"):
		r.metricHandler.ServeHTTP(w, req)
	case path == "/api/snmp/collect" || HasPrefix(path, "/api/snmp/collect/"):
		r.snmpCollectHandler.ServeHTTP(w, req)
	case path == "/api/audit" || HasPrefix(path, "/api/audit/"):
		r.auditHandler.ServeHTTP(w, req)
	case path == "/api/auth" || HasPrefix(path, "/api/auth/"):
		r.authHandler.ServeHTTP(w, req)
	case path == "/api/users" || HasPrefix(path, "/api/users/"):
		r.userHandler.ServeHTTP(w, req)
	case path == "/api/tickets" || HasPrefix(path, "/api/tickets/"):
		r.ticketHandler.ServeHTTP(w, req)
	case path == "/api/comap/readings" || HasPrefix(path, "/api/comap/readings/"):
		r.comapReadingHandler.ServeHTTP(w, req)
	case path == "/api/discovered-devices" || HasPrefix(path, "/api/discovered-devices/"):
		r.discoveredDeviceHandler.ServeHTTP(w, req)
	case path == "/api/regions" || HasPrefix(path, "/api/regions/"):
		r.regionHandler.ServeHTTP(w, req)
	case path == "/api/operators" || HasPrefix(path, "/api/operators/"):
		r.operatorHandler.ServeHTTP(w, req)
	case path == "/api/sla" || HasPrefix(path, "/api/sla/"):
		r.slaHandler.ServeHTTP(w, req)
	case path == "/api/radio-kpi" || HasPrefix(path, "/api/radio-kpi/"):
		r.radioKPIHandler.ServeHTTP(w, req)
	case path == "/api/backhaul" || HasPrefix(path, "/api/backhaul/"): // <-- ROTAS DE BACKHAUL
		r.backhaulInterfaceHandler.ServeHTTP(w, req)
	case path == "/api/site-environment" || HasPrefix(path, "/api/site-environment/"): // <-- ROTAS DE AMBIENTE
		r.siteEnvironmentHandler.ServeHTTP(w, req)
	default:
		http.NotFound(w, req)
	}
}

// Funções auxiliares
func HasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}