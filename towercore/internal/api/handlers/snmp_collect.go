package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"towercore/internal/core/services"
	"towercore/pkg/apierror"
)

type SNMPCollectHandler struct {
	service *services.SNMPIngestService
}

func NewSNMPCollectHandler(service *services.SNMPIngestService) *SNMPCollectHandler {
	return &SNMPCollectHandler{service: service}
}

type snmpCollectRequest struct {
	TowerID     string             `json:"tower_id"`
	Vendor      string             `json:"vendor"`
	CollectedAt string             `json:"collected_at"`
	Samples     map[string]float64 `json:"samples"`
}

func (h *SNMPCollectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		apierror.MethodNotAllowed(w)
		return
	}

	var req snmpCollectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, "invalid json payload")
		return
	}

	snapshot := services.SNMPSnapshot{
		TowerID: strings.TrimSpace(req.TowerID),
		Vendor:  strings.TrimSpace(req.Vendor),
		Samples: req.Samples,
	}
	if req.CollectedAt != "" {
		parsed, err := time.Parse(time.RFC3339, req.CollectedAt)
		if err != nil {
			apierror.BadRequest(w, "collected_at must be RFC3339")
			return
		}
		snapshot.CollectedAt = parsed
	}

	if err := h.service.Ingest(r.Context(), snapshot); err != nil {
		apierror.BadRequest(w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "accepted",
		"vendor": strings.ToLower(strings.TrimSpace(req.Vendor)),
	})
}
