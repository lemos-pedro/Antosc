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

