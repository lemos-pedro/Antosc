package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/antosc/aip/internal/api/dto"
	"github.com/antosc/aip/internal/repository/postgres"
)

type ConsumptionNormHandler struct {
	repo postgres.ConsumptionNormRepository
}

func NewConsumptionNormHandler(repo postgres.ConsumptionNormRepository) *ConsumptionNormHandler {
	return &ConsumptionNormHandler{repo: repo}
}

// Create — POST /api/v1/norms  (só o Controller deve ter permissão para chamar isto;
// a validação de role assume-se feita no middleware de auth, não aqui).
func (h *ConsumptionNormHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.ConsumptionNormRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "corpo inválido", http.StatusBadRequest)
		return
	}
	if req.TowerID == "" || req.EquipmentType == "" || req.Unit == "" || req.DefinedBy == "" {
		http.Error(w, "tower_id, equipment_type, unit e defined_by são obrigatórios", http.StatusBadRequest)
		return
	}
	if req.TolerancePercent == 0 {
		req.TolerancePercent = 10.0
	}

	n, err := h.repo.Create(r.Context(), postgres.ConsumptionNorm{
		TowerID:          req.TowerID,
		EquipmentType:    req.EquipmentType,
		ExpectedValue:    req.ExpectedValue,
		Unit:             req.Unit,
		TolerancePercent: req.TolerancePercent,
		DefinedBy:        req.DefinedBy,
	})
	if err != nil {
		http.Error(w, "erro ao gravar norma", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, toNormResponse(n))
}

// Update — PUT /api/v1/norms/{id}
func (h *ConsumptionNormHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req dto.ConsumptionNormRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "corpo inválido", http.StatusBadRequest)
		return
	}

	n, err := h.repo.Update(r.Context(), id, postgres.ConsumptionNorm{
		ExpectedValue:    req.ExpectedValue,
		Unit:             req.Unit,
		TolerancePercent: req.TolerancePercent,
		DefinedBy:        req.DefinedBy,
	})
	if err != nil {
		http.Error(w, "erro ao atualizar norma", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, toNormResponse(n))
}

// ListByTower — GET /api/v1/norms/{tower_id}
func (h *ConsumptionNormHandler) ListByTower(w http.ResponseWriter, r *http.Request) {
	towerID := r.PathValue("tower_id")

	norms, err := h.repo.ListByTower(r.Context(), towerID)
	if err != nil {
		http.Error(w, "erro ao listar normas", http.StatusInternalServerError)
		return
	}

	out := make([]dto.ConsumptionNormResponse, 0, len(norms))
	for _, n := range norms {
		out = append(out, toNormResponse(n))
	}
	writeJSON(w, http.StatusOK, out)
}

// Deactivate — DELETE /api/v1/norms/{id}
func (h *ConsumptionNormHandler) Deactivate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.repo.Deactivate(r.Context(), id); err != nil {
		http.Error(w, "erro ao desativar norma", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func toNormResponse(n postgres.ConsumptionNorm) dto.ConsumptionNormResponse {
	return dto.ConsumptionNormResponse{
		ID:               n.ID,
		TowerID:          n.TowerID,
		EquipmentType:    n.EquipmentType,
		ExpectedValue:    n.ExpectedValue,
		Unit:             n.Unit,
		TolerancePercent: n.TolerancePercent,
		DefinedBy:        n.DefinedBy,
		Active:           n.Active,
		CreatedAt:        n.CreatedAt,
		UpdatedAt:        n.UpdatedAt,
	}
}
