package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
	"towercore/internal/core/services"
	"towercore/pkg/apierror"
)

type MetricHandler struct {
	service *services.MetricService
}

func NewMetricHandler(service *services.MetricService) *MetricHandler {
	return &MetricHandler{service: service}
}

type createMetricRequest struct {
	ID          string             `json:"metric_id"`
	TowerID     string             `json:"tower_id"`
	CollectedAt string             `json:"collected_at"`
	Values      map[string]float64 `json:"metrics"`
}

func (h *MetricHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.create(w, r)
	case http.MethodGet:
		h.list(w, r)
	default:
		apierror.MethodNotAllowed(w)
	}
}

func (h *MetricHandler) create(w http.ResponseWriter, r *http.Request) {
	var req createMetricRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, "invalid json payload")
		return
	}

	metric := domain.Metric{
		ID:      strings.TrimSpace(req.ID),
		TowerID: strings.TrimSpace(req.TowerID),
		Values:  req.Values,
	}

	if req.CollectedAt != "" {
		parsed, err := time.Parse(time.RFC3339, req.CollectedAt)
		if err != nil {
			apierror.BadRequest(w, "collected_at must be RFC3339")
			return
		}
		metric.CollectedAt = parsed
	}

	if err := h.service.Create(r.Context(), &metric); err != nil {
		apierror.BadRequest(w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(metric)
}

func (h *MetricHandler) list(w http.ResponseWriter, r *http.Request) {
	filter := interfaces.MetricFilter{
		TowerID: r.URL.Query().Get("tower_id"),
		Limit:   parseIntDefault(r.URL.Query().Get("limit"), 50),
		Offset:  parseIntDefault(r.URL.Query().Get("offset"), 0),
	}

	if rawFrom := strings.TrimSpace(r.URL.Query().Get("from")); rawFrom != "" {
		from, err := time.Parse(time.RFC3339, rawFrom)
		if err != nil {
			apierror.BadRequest(w, "from must be RFC3339")
			return
		}
		filter.From = &from
	}
	if rawTo := strings.TrimSpace(r.URL.Query().Get("to")); rawTo != "" {
		to, err := time.Parse(time.RFC3339, rawTo)
		if err != nil {
			apierror.BadRequest(w, "to must be RFC3339")
			return
		}
		filter.To = &to
	}

	metrics, total, err := h.service.List(r.Context(), filter)
	if err != nil {
		apierror.Internal(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(metrics)
}
