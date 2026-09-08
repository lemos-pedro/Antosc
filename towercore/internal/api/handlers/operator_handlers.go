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

type OperatorHandler struct {
	service *services.OperatorService
}

func NewOperatorHandler(service *services.OperatorService) *OperatorHandler {
	return &OperatorHandler{service: service}
}

func (h *OperatorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.list(w, r)
	case http.MethodPost:
		h.create(w, r)
	default:
		apierror.MethodNotAllowed(w)
	}
}

type createOperatorRequest struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

func (h *OperatorHandler) list(w http.ResponseWriter, r *http.Request) {
	operators, err := h.service.List(r.Context())
	if err != nil {
		apierror.Internal(w)
		return
	}

	if operators == nil {
		operators = []domain.Operator{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(operators)
}

func (h *OperatorHandler) create(w http.ResponseWriter, r *http.Request) {
	var req createOperatorRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierror.BadRequest(w, "invalid json payload")
		return
	}

	operator := domain.Operator{
		Name:      strings.TrimSpace(req.Name),
		Code:      strings.TrimSpace(req.Code),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := h.service.Save(r.Context(), &operator); err != nil {
		apierror.BadRequest(w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(operator)
}

type updateOperatorRequest struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

// ServeByID trata GET/PUT/DELETE /api/v1/operators/{id} — registado
// separadamente no router porque usa um path param, ao contrário de
// GET/POST /api/v1/operators (sem id).
func (h *OperatorHandler) ServeByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		apierror.BadRequest(w, "operator id is required in path")
		return
	}

	switch r.Method {
	case http.MethodGet:
		operator, err := h.service.GetByID(r.Context(), id)
		if err != nil {
			h.writeOperatorError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(operator)

	case http.MethodPut:
		var req updateOperatorRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierror.BadRequest(w, "invalid json payload")
			return
		}
		operator := domain.Operator{
			OperatorID: id,
			Name:       strings.TrimSpace(req.Name),
			Code:       strings.TrimSpace(req.Code),
		}
		if err := h.service.Update(r.Context(), &operator); err != nil {
			h.writeOperatorError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(operator)

	case http.MethodDelete:
		if err := h.service.Delete(r.Context(), id); err != nil {
			h.writeOperatorError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		apierror.MethodNotAllowed(w)
	}
}

func (h *OperatorHandler) writeOperatorError(w http.ResponseWriter, err error) {
	if err == interfaces.ErrOperatorNotFound {
		apierror.Write(w, http.StatusNotFound, "resource_not_found", "operator not found")
		return
	}
	if strings.Contains(err.Error(), "foreign key") || strings.Contains(err.Error(), "violates") {
		apierror.Write(w, http.StatusConflict, "conflict", "operator has towers assigned and cannot be deleted")
		return
	}
	apierror.Internal(w)
}
