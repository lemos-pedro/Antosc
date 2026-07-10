package handlers

import (
	"encoding/json"
	"net/http"

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