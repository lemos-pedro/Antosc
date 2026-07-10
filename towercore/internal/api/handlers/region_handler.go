package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"towercore/internal/core/domain"
	"towercore/internal/core/services"
	"towercore/pkg/apierror"
)

type RegionHandler struct {
	service *services.RegionService
}

func NewRegionHandler(service *services.RegionService) *RegionHandler {
	return &RegionHandler{service: service}
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