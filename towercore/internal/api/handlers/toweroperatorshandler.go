package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"towercore/internal/core/interfaces"
	"towercore/internal/core/services"
	"towercore/pkg/apierror"
)

type TowerOperatorHandler struct {
	service *services.TowerService
}

func NewTowerOperatorHandler(service *services.TowerService) *TowerOperatorHandler {
	return &TowerOperatorHandler{service: service}
}

func (h *TowerOperatorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.add(w, r)
	case http.MethodDelete:
		h.remove(w, r)
	default:
		apierror.MethodNotAllowed(w)
	}
}

type addOperatorRequest struct {
	OperatorID string `json:"operator_id"`
}

func (h *TowerOperatorHandler) add(w http.ResponseWriter, r *http.Request) {
	towerID := strings.TrimSpace(r.PathValue("id"))
	if towerID == "" {
		apierror.BadRequest(w, "tower id is required in path")
		return
	}

	var req addOperatorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, "invalid json payload")
		return
	}
	if strings.TrimSpace(req.OperatorID) == "" {
		apierror.BadRequest(w, "operator_id is required")
		return
	}

	if err := h.service.AddOperator(r.Context(), towerID, req.OperatorID); err != nil {
		if errors.Is(err, interfaces.ErrTowerNotFound) {
			apierror.Write(w, http.StatusNotFound, "resource_not_found", "tower not found")
			return
		}
		apierror.BadRequest(w, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *TowerOperatorHandler) remove(w http.ResponseWriter, r *http.Request) {
	towerID := strings.TrimSpace(r.PathValue("id"))
	operatorID := strings.TrimSpace(r.PathValue("operator_id"))
	if towerID == "" || operatorID == "" {
		apierror.BadRequest(w, "tower id and operator id are required in path")
		return
	}

	if err := h.service.RemoveOperator(r.Context(), towerID, operatorID); err != nil {
		if errors.Is(err, interfaces.ErrOperatorNotFound) {
			apierror.Write(w, http.StatusNotFound, "resource_not_found", "operator association not found")
			return
		}
		apierror.BadRequest(w, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}