package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/antosc/aip/internal/api/dto"
	"github.com/antosc/aip/internal/repository/postgres"
)

type IncidentCauseHandler struct {
	repo postgres.IncidentCauseRepository
}

func NewIncidentCauseHandler(repo postgres.IncidentCauseRepository) *IncidentCauseHandler {
	return &IncidentCauseHandler{repo: repo}
}

// Confirm — POST /api/v1/incidents/{id}/confirm
// O&M confirma ou corrige a causa prevista pelo ML. Se a causa confirmada for
// diferente da prevista, o repository marca automaticamente como "corrected" —
// este sinal é o que vai alimentar o retreino do modelo de causas mais tarde.
func (h *IncidentCauseHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req dto.IncidentCauseConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "corpo inválido", http.StatusBadRequest)
		return
	}
	if req.ConfirmedCause == "" || req.ConfirmedBy == "" {
		http.Error(w, "confirmed_cause e confirmed_by são obrigatórios", http.StatusBadRequest)
		return
	}

	c, err := h.repo.Confirm(r.Context(), id, req.ConfirmedCause, req.ConfirmedBy)
	if err != nil {
		http.Error(w, "erro ao confirmar causa", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, toIncidentCauseResponse(c))
}

// PendingConfirmation — GET /api/v1/incidents/pending
// Lista de incidentes que o O&M ainda não confirmou/corrigiu.
func (h *IncidentCauseHandler) PendingConfirmation(w http.ResponseWriter, r *http.Request) {
	list, err := h.repo.ListPendingConfirmation(r.Context())
	if err != nil {
		http.Error(w, "erro ao listar pendentes", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, toIncidentCauseResponses(list))
}

// ByPeriod — GET /api/v1/incidents?from=2026-07-01&to=2026-07-08
// Base do relatório semanal: sites que caíram no intervalo + causa (ML e/ou confirmada).
func (h *IncidentCauseHandler) ByPeriod(w http.ResponseWriter, r *http.Request) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		http.Error(w, "parâmetro 'from' inválido, use YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		// Sem 'to', assume-se 7 dias após 'from' (relatório semanal por omissão).
		to = from.AddDate(0, 0, 7)
	}

	list, err := h.repo.ListByPeriod(r.Context(), from, to)
	if err != nil {
		http.Error(w, "erro ao listar incidentes", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, toIncidentCauseResponses(list))
}

// ByTower — GET /api/v1/incidents/tower/{tower_id}
func (h *IncidentCauseHandler) ByTower(w http.ResponseWriter, r *http.Request) {
	towerID := r.PathValue("tower_id")

	list, err := h.repo.ListByTower(r.Context(), towerID, 100)
	if err != nil {
		http.Error(w, "erro ao listar incidentes da torre", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, toIncidentCauseResponses(list))
}

func toIncidentCauseResponse(c postgres.IncidentCause) dto.IncidentCauseResponse {
	resp := dto.IncidentCauseResponse{
		ID:                c.ID,
		TowerID:           c.TowerID,
		IncidentStartedAt: c.IncidentStartedAt,
		Status:            c.Status,
		CreatedAt:         c.CreatedAt,
	}
	if c.EventID.Valid {
		resp.EventID = &c.EventID.String
	}
	if c.IncidentEndedAt.Valid {
		resp.IncidentEndedAt = &c.IncidentEndedAt.Time
	}
	if c.MLPredictedCause.Valid {
		resp.MLPredictedCause = &c.MLPredictedCause.String
	}
	if c.MLConfidence.Valid {
		resp.MLConfidence = &c.MLConfidence.Float64
	}
	if c.ConfirmedCause.Valid {
		resp.ConfirmedCause = &c.ConfirmedCause.String
	}
	if c.ConfirmedBy.Valid {
		resp.ConfirmedBy = &c.ConfirmedBy.String
	}
	if c.ConfirmedAt.Valid {
		resp.ConfirmedAt = &c.ConfirmedAt.Time
	}
	return resp
}

func toIncidentCauseResponses(list []postgres.IncidentCause) []dto.IncidentCauseResponse {
	out := make([]dto.IncidentCauseResponse, 0, len(list))
	for _, c := range list {
		out = append(out, toIncidentCauseResponse(c))
	}
	return out
}
