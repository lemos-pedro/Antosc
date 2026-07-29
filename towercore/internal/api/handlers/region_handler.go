package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
	"towercore/internal/core/services"
	"towercore/pkg/apierror"
)

type RegionHandler struct {
	service    *services.RegionService
	slaService *services.SLAService
}

func NewRegionHandler(service *services.RegionService, slaService *services.SLAService) *RegionHandler {
	return &RegionHandler{service: service, slaService: slaService}
}

func (h *RegionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.list(w, r)
	case http.MethodPost:
		h.create(w, r)
	default:
		apierror.MethodNotAllowed(w)
	}
}

type createRegionRequest struct {
	Name string `json:"name"`
}

func (h *RegionHandler) list(w http.ResponseWriter, r *http.Request) {
	regions, err := h.service.List(r.Context())
	if err != nil {
		apierror.Internal(w)
		return
	}

	if regions == nil {
		regions = []domain.Region{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(regions)
}

func (h *RegionHandler) create(w http.ResponseWriter, r *http.Request) {
	var req createRegionRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, "invalid json payload")
		return
	}

	region := domain.Region{
		Name:      strings.TrimSpace(req.Name),
		CreatedAt: time.Now().UTC(),
	}

	if err := h.service.Save(r.Context(), &region); err != nil {
		apierror.BadRequest(w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(region)
}

type regionDetailResponse struct {
	RegionID      string  `json:"region_id"`
	Name          string  `json:"name"`
	CreatedAt     any     `json:"created_at"`
	Availability  float64 `json:"availability_percent"`
	TotalSites    int     `json:"total_sites"`
	OnlineSites   int     `json:"online_sites"`
	OfflineSites  int     `json:"offline_sites"`
	DegradedSites int     `json:"degraded_sites"`
}

// ServeByID trata GET /api/v1/regions/{id} — detalhe com métricas agregadas
// (reaproveita SLAService.GetByRegion), registado separadamente no router
// porque usa path param.
func (h *RegionHandler) ServeByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		apierror.MethodNotAllowed(w)
		return
	}

	id := r.PathValue("id")
	if id == "" {
		apierror.BadRequest(w, "region id is required in path")
		return
	}

	region, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if err == interfaces.ErrRegionNotFound {
			apierror.Write(w, http.StatusNotFound, "resource_not_found", "region not found")
			return
		}
		apierror.Internal(w)
		return
	}

	resp := regionDetailResponse{
		RegionID:  region.RegionID,
		Name:      region.Name,
		CreatedAt: region.CreatedAt,
	}

	if sla, err := h.slaService.GetByRegion(r.Context(), id); err == nil && sla != nil {
		resp.Availability = sla.Availability
		resp.TotalSites = sla.TotalSites
		resp.OnlineSites = sla.OnlineSites
		resp.OfflineSites = sla.OfflineSites
		resp.DegradedSites = sla.DegradedSites
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}