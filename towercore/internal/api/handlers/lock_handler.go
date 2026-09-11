// internal/api/handlers/lock_handler.go
package handlers

import (
	"net/http"

	"towercore/internal/adapters/hizima"
	"towercore/internal/core/interfaces"
	"towercore/pkg/apierror"
)

// LockHandler expõe os dados de lock/eventos/work-orders da Hizima ao dashboard.
// Não persiste nada — cada pedido chama a API da Hizima ao vivo (read-only).
type LockHandler struct {
	Client *hizima.Client
	// ResolveStationNo resolve tower_id (UUID interno) -> StationNo (sno) da Hizima.
	// Pode ser um map[string]string estático vindo de config; não implica tabela nova.
	ResolveStationNo func(towerID string) (string, bool)
}

func NewLockHandler(client *hizima.Client, resolveStationNo func(towerID string) (string, bool)) *LockHandler {
	return &LockHandler{Client: client, ResolveStationNo: resolveStationNo}
}


// GetStatus atende GET /api/v1/towers/{tower_id}/lock-status
func (h *LockHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	towerID := r.PathValue("tower_id")
	stationNo, ok := h.ResolveStationNo(towerID)
	if !ok {
		apierror.Write(w, http.StatusNotFound, "resource_not_found", "tower has no associated Hizima station")
		return
	}

	result, err := h.Client.GetLockStatus(r.Context(),
		interfaces.LockStatusFilter{StationNo: stationNo},
		interfaces.PageRequest{Current: 1, Size: 20},
	)
	if err != nil {
		apierror.Write(w, http.StatusBadGateway, "internal_error", "failed to fetch lock status from Hizima")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": result.Records,
		"meta": map[string]int{"total": result.Total, "current": result.Current, "size": result.Size},
	})
}

// GetEvents atende GET /api/v1/towers/{tower_id}/lock-events
func (h *LockHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	towerID := r.PathValue("tower_id")
	stationNo, ok := h.ResolveStationNo(towerID)
	if !ok {
		apierror.Write(w, http.StatusNotFound, "resource_not_found", "tower has no associated Hizima station")
		return
	}

	result, err := h.Client.GetLockEvents(r.Context(),
		interfaces.LockEventFilter{StationNo: stationNo},
		interfaces.PageRequest{Current: 1, Size: 50},
	)
	if err != nil {
		apierror.Write(w, http.StatusBadGateway, "internal_error", "failed to fetch lock events from Hizima")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": result.Records,
		"meta": map[string]int{"total": result.Total, "current": result.Current, "size": result.Size},
	})
}

// GetWorkOrders atende GET /api/v1/towers/{tower_id}/work-orders
func (h *LockHandler) GetWorkOrders(w http.ResponseWriter, r *http.Request) {
	towerID := r.PathValue("tower_id")
	stationNo, ok := h.ResolveStationNo(towerID)
	if !ok {
		apierror.Write(w, http.StatusNotFound, "resource_not_found", "tower has no associated Hizima station")
		return
	}

	result, err := h.Client.GetWorkOrders(r.Context(),
		interfaces.WorkOrderFilter{StationName: stationNo},
		interfaces.PageRequest{Current: 1, Size: 20},
	)
	if err != nil {
		apierror.Write(w, http.StatusBadGateway, "internal_error", "failed to fetch work orders from Hizima")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": result.Records,
		"meta": map[string]int{"total": result.Total, "current": result.Current, "size": result.Size},
	})
}