package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"towercore/internal/core/interfaces"
	"towercore/internal/core/services"
	"towercore/pkg/apierror"
)

// DiscoveredDeviceHandler expõe os dispositivos encontrados por
// network discovery SNMP para revisão e promoção manual.
//
// Rotas esperadas (ver routes/router.go):
//
//	GET   /discovered-devices            -> list
//	POST  /discovered-devices/{id}/promote -> promote
//	POST  /discovered-devices/{id}/ignore  -> ignore
type DiscoveredDeviceHandler struct {
	repo      interfaces.DiscoveredDeviceRepository
	promotion *services.DiscoveredDevicePromotionService
}

func NewDiscoveredDeviceHandler(
	repo interfaces.DiscoveredDeviceRepository,
	promotion *services.DiscoveredDevicePromotionService,
) *DiscoveredDeviceHandler {
	return &DiscoveredDeviceHandler{repo: repo, promotion: promotion}
}

func (h *DiscoveredDeviceHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		apierror.MethodNotAllowed(w)
		return
	}

	filter := interfaces.DiscoveredDeviceFilter{
		Status: interfaces.DiscoveredDeviceStatusFilter(r.URL.Query().Get("status")),
		Limit:  parseIntDefault(r.URL.Query().Get("limit"), 100),
		Offset: parseIntDefault(r.URL.Query().Get("offset"), 0),
	}

	devices, total, err := h.repo.List(r.Context(), filter)
	if err != nil {
		apierror.Internal(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(devices)
}

type promoteDeviceRequest struct {
	Name       string `json:"name"`
	OperatorID string `json:"operator_id"`
	RegionID   string `json:"region_id"`
	Vendor     string `json:"vendor"`
}

func (h *DiscoveredDeviceHandler) Promote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		apierror.MethodNotAllowed(w)
		return
	}

	deviceID := strings.TrimSpace(r.PathValue("id"))
	if deviceID == "" {
		apierror.BadRequest(w, "device id is required in path")
		return
	}

	var req promoteDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, "invalid json payload")
		return
	}

	tower, err := h.promotion.Promote(r.Context(), deviceID, services.PromotionInput{
		Name:       req.Name,
		OperatorID: req.OperatorID,
		RegionID:   req.RegionID,
		Vendor:     req.Vendor,
	})
	if err != nil {
		switch {
		case errors.Is(err, interfaces.ErrDiscoveredDeviceNotFound):
			apierror.Write(w, http.StatusNotFound, "resource_not_found", "discovered device not found")
		default:
			apierror.BadRequest(w, err.Error())
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(tower)
}

func (h *DiscoveredDeviceHandler) Ignore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		apierror.MethodNotAllowed(w)
		return
	}

	deviceID := strings.TrimSpace(r.PathValue("id"))
	if deviceID == "" {
		apierror.BadRequest(w, "device id is required in path")
		return
	}

	if err := h.promotion.Ignore(r.Context(), deviceID); err != nil {
		switch {
		case errors.Is(err, interfaces.ErrDiscoveredDeviceNotFound):
			apierror.Write(w, http.StatusNotFound, "resource_not_found", "discovered device not found")
		default:
			apierror.BadRequest(w, err.Error())
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ServeHTTP permite usar DiscoveredDeviceHandler como http.Handler.
func (h *DiscoveredDeviceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.List(w, r)
	case http.MethodPost:
		// Determinar ação baseada na suffix da rota: /{id}/promote ou /{id}/ignore
		if strings.HasSuffix(r.URL.Path, "/promote") {
			h.Promote(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/ignore") {
			h.Ignore(w, r)
			return
		}
		apierror.MethodNotAllowed(w)
	default:
		apierror.MethodNotAllowed(w)
	}
}
