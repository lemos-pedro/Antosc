package routes

import (
	"net/http"

	"towercore/internal/api/handlers"
)

// RegisterNetworkLinkRoutes regista os endpoints de /api/v1/links no mux
// principal da aplicação. Chamar a partir do setup de rotas já existente
// em cmd/api/main.go, junto com o registo das rotas de towers/operators.
func RegisterNetworkLinkRoutes(mux *http.ServeMux, h *handlers.NetworkLinksHandler) {
	mux.HandleFunc("GET /api/v1/links", h.ListLinks)
	mux.HandleFunc("GET /api/v1/links/{link_id}", h.GetLink)
	mux.HandleFunc("GET /api/v1/links/{link_id}/events", h.GetLinkEvents)
}

// RegisterZabbixLinkRoutes adds the diagnostic and synchronization endpoints
// used to import IF-MIB values already collected by Zabbix.
func RegisterZabbixLinkRoutes(mux *http.ServeMux, h *handlers.ZabbixLinksHandler) {
	mux.HandleFunc("GET /api/v1/links/zabbix/items", h.Inspect)
	mux.HandleFunc("POST /api/v1/links/sync/zabbix", h.Sync)
}
