package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"towercore/internal/api/middleware"
	"towercore/internal/core/interfaces"
	"towercore/internal/core/services"
	"towercore/pkg/apierror"
)

type TicketHandler struct {
	service *services.TicketService
}

func NewTicketHandler(service *services.TicketService) *TicketHandler {
	return &TicketHandler{service: service}
}

func (h *TicketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet:
		h.list(w, r)
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/ack"):
		h.ack(w, r)
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/close"):
		h.close(w, r)
	default:
		apierror.MethodNotAllowed(w)
	}
}

func (h *TicketHandler) list(w http.ResponseWriter, r *http.Request) {
	filter := interfaces.TicketFilter{
		Status:  r.URL.Query().Get("status"),
		TowerID: r.URL.Query().Get("tower_id"),
		Limit:   parseIntDefault(r.URL.Query().Get("limit"), 50),
		Offset:  parseIntDefault(r.URL.Query().Get("offset"), 0),
	}

	tickets, total, err := h.service.List(r.Context(), filter)
	if err != nil {
		apierror.Internal(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(tickets)
}

func (h *TicketHandler) ack(w http.ResponseWriter, r *http.Request) {
	ticketID := strings.TrimSpace(r.PathValue("ticket_id"))
	if ticketID == "" {
		apierror.BadRequest(w, "ticket id is required in path")
		return
	}

	ticket, err := h.service.Acknowledge(r.Context(), middleware.UserIDFromContext(r.Context()), ticketID)
	if err != nil {
		switch {
		case errors.Is(err, interfaces.ErrTicketNotFound):
			apierror.Write(w, http.StatusNotFound, "resource_not_found", "ticket not found")
		default:
			apierror.BadRequest(w, err.Error())
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(ticket)
}

func (h *TicketHandler) close(w http.ResponseWriter, r *http.Request) {
	ticketID := strings.TrimSpace(r.PathValue("ticket_id"))
	if ticketID == "" {
		apierror.BadRequest(w, "ticket id is required in path")
		return
	}

	ticket, err := h.service.Close(r.Context(), middleware.UserIDFromContext(r.Context()), ticketID)
	if err != nil {
		switch {
		case errors.Is(err, interfaces.ErrTicketNotFound):
			apierror.Write(w, http.StatusNotFound, "resource_not_found", "ticket not found")
		default:
			apierror.BadRequest(w, err.Error())
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(ticket)
}
