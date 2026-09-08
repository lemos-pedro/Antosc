package handlers

import (
	"encoding/json"
	"net/http"

	"towercore/internal/core/interfaces"
	"towercore/internal/core/services"
	"towercore/pkg/apierror"
)

type SLAHandler struct {
	service *services.SLAService
}

func NewSLAHandler(service *services.SLAService) *SLAHandler {
	return &SLAHandler{service: service}
}

func (h *SLAHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.global(w, r)
	default:
		apierror.MethodNotAllowed(w)
	}
}

func (h *SLAHandler) global(w http.ResponseWriter, r *http.Request) {
	sla, err := h.service.GetGlobal(r.Context())
	if err != nil {
		apierror.Internal(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(sla)
}

// ServeRegion trata GET /api/v1/sla/region/{id} — registado separadamente
// no router porque usa um path param, ao contrário de /sla/global.
func (h *SLAHandler) ServeRegion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		apierror.MethodNotAllowed(w)
		return
	}

	regionID := r.PathValue("id")
	if regionID == "" {
		apierror.BadRequest(w, "region id is required")
		return
	}

	sla, err := h.service.GetByRegion(r.Context(), regionID)
	if err != nil {
		if err == interfaces.ErrRegionNotFound {
			apierror.Write(w, http.StatusNotFound, "resource_not_found", "region not found")
			return
		}
		apierror.Internal(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(sla)
}
