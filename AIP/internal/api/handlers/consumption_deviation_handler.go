package handlers

import (
	"net/http"
	"time"

	"github.com/antosc/aip/internal/api/dto"
	"github.com/antosc/aip/internal/services/consumption"
)

type ConsumptionDeviationHandler struct {
	service consumption.Service
}

func NewConsumptionDeviationHandler(service consumption.Service) *ConsumptionDeviationHandler {
	return &ConsumptionDeviationHandler{service: service}
}

// ByTower — GET /api/v1/consumption/deviations/{tower_id}?from=2026-07-01&to=2026-07-08
// Compara o consumo real registado no período com a norma definida pelo Controller.
// Sem 'to', assume-se o período de uma semana a partir de 'from' (uso pelo relatório semanal).
func (h *ConsumptionDeviationHandler) ByTower(w http.ResponseWriter, r *http.Request) {
	towerID := r.PathValue("tower_id")

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		http.Error(w, "parâmetro 'from' inválido, use YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		to = from.AddDate(0, 0, 7)
	}

	deviations, err := h.service.ComputeForTower(r.Context(), towerID, from, to)
	if err != nil {
		http.Error(w, "erro ao calcular desvios de consumo", http.StatusInternalServerError)
		return
	}

	out := make([]dto.ConsumptionDeviationResponse, 0, len(deviations))
	for _, d := range deviations {
		out = append(out, dto.ConsumptionDeviationResponse{
			TowerID:          d.TowerID,
			EquipmentType:    d.EquipmentType,
			ExpectedValue:    d.ExpectedValue,
			AverageActual:    d.AverageActual,
			Unit:             d.Unit,
			TolerancePercent: d.TolerancePercent,
			DeviationPercent: d.DeviationPercent,
			WithinNorm:       d.WithinNorm,
			SampleCount:      d.SampleCount,
		})
	}

	writeJSON(w, http.StatusOK, out)
}

// Compare — GET /api/v1/consumption/deviations/{tower_id}/compare
//   ?period_a_from=2026-06-01&period_a_to=2026-06-30&period_b_from=2026-07-01&period_b_to=2026-07-19
// Compara o consumo médio de dois períodos diferentes (ex: mês passado vs este mês).
func (h *ConsumptionDeviationHandler) Compare(w http.ResponseWriter, r *http.Request) {
	towerID := r.PathValue("tower_id")
	q := r.URL.Query()

	aFrom, err := time.Parse("2006-01-02", q.Get("period_a_from"))
	if err != nil {
		http.Error(w, "parâmetro 'period_a_from' inválido, use YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	aTo, err := time.Parse("2006-01-02", q.Get("period_a_to"))
	if err != nil {
		aTo = aFrom.AddDate(0, 0, 7)
	}
	bFrom, err := time.Parse("2006-01-02", q.Get("period_b_from"))
	if err != nil {
		http.Error(w, "parâmetro 'period_b_from' inválido, use YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	bTo, err := time.Parse("2006-01-02", q.Get("period_b_to"))
	if err != nil {
		bTo = bFrom.AddDate(0, 0, 7)
	}

	comparisons, err := h.service.Compare(r.Context(), towerID, aFrom, aTo, bFrom, bTo)
	if err != nil {
		http.Error(w, "erro ao comparar períodos", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, comparisons)
}
