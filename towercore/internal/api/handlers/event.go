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

type EventHandler struct {
	service *services.EventService
}

func NewEventHandler(service *services.EventService) *EventHandler {
	return &EventHandler{service: service}
}

type createEventRequest struct {
	ID         string `json:"event_id"`
	TowerID    string `json:"tower_id"`
	Type       string `json:"type"`
	Severity   string `json:"severity"`
	Message    string `json:"message"`
	OccurredAt string `json:"occurred_at"`
}

func (h *EventHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.create(w, r)
	case http.MethodGet:
		h.list(w, r)
	default:
		apierror.MethodNotAllowed(w)
	}
}

func (h *EventHandler) create(w http.ResponseWriter, r *http.Request) {
	var req createEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, "invalid json payload")
		return
	}

	event := domain.Event{
		ID:       strings.TrimSpace(req.ID),
		TowerID:  strings.TrimSpace(req.TowerID),
		Type:     domain.EventType(strings.TrimSpace(req.Type)),
		Severity: domain.EventSeverity(strings.TrimSpace(req.Severity)),
		Message:  strings.TrimSpace(req.Message),
	}

	if req.OccurredAt != "" {
		parsed, err := time.Parse(time.RFC3339, req.OccurredAt)
		if err != nil {
			apierror.BadRequest(w, "occurred_at must be RFC3339")
			return
		}
		event.OccurredAt = parsed
	}

	if err := h.service.Create(r.Context(), &event); err != nil {
		apierror.BadRequest(w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(event)
}

func (h *EventHandler) ListByTower(w http.ResponseWriter, r *http.Request) {
	towerID := r.PathValue("id")

	filter := interfaces.EventFilter{
		TowerID:  towerID,
		Type:     r.URL.Query().Get("type"),
		Severity: r.URL.Query().Get("severity"),
		Status:   r.URL.Query().Get("status"),
		Limit:    parseIntDefault(r.URL.Query().Get("limit"), 50),
		Offset:   parseIntDefault(r.URL.Query().Get("offset"), 0),
	}

	events, total, err := h.service.List(r.Context(), filter)
	if err != nil {
		apierror.Internal(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(events)
}

func (h *EventHandler) list(w http.ResponseWriter, r *http.Request) {
	// CORRIGIDO (ponto 3): faltava repassar "status" da query string aqui.
	// Já existia no ListByTower, mas este list() genérico (GET /events)
	// continuava a ignorar o filtro, devolvendo sempre eventos resolvidos
	// e abertos misturados a quem chamasse este endpoint sem passar por
	// /towers/{id}/events.
	filter := interfaces.EventFilter{
		TowerID:  r.URL.Query().Get("tower_id"),
		Type:     r.URL.Query().Get("type"),
		Severity: r.URL.Query().Get("severity"),
		Status:   r.URL.Query().Get("status"),
		Limit:    parseIntDefault(r.URL.Query().Get("limit"), 50),
		Offset:   parseIntDefault(r.URL.Query().Get("offset"), 0),
	}

	events, total, err := h.service.List(r.Context(), filter)
	if err != nil {
		apierror.Internal(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(events)
}