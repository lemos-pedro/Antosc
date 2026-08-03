package handlers

import (
	"net/http"
	"time"

	"github.com/antosc/aip/internal/repository/postgres"
	"github.com/antosc/aip/internal/services/consumption"
)

// PowerBIHandler expõe endpoints read-only optimizados para o Power BI
// (ou qualquer ferramenta BI). Formato JSON tabular, estável — o Power BI
// faz refresh periódico via "Obter dados → Web" ou gateway on-premises.
//
// Catálogo:
//   GET /api/v1/powerbi/catalog
//   GET /api/v1/powerbi/incidents?from=&to=
//   GET /api/v1/powerbi/kpis?from=&to=
//   GET /api/v1/powerbi/consumption?from=&to=
//   GET /api/v1/powerbi/predictions
//   GET /api/v1/powerbi/towers
type PowerBIHandler struct {
	incidents   postgres.IncidentCauseRepository
	predictions postgres.PredictionRepository
	norms       postgres.ConsumptionNormRepository
	deviations  consumption.Service
	towers      postgres.TowerRepository
}

func NewPowerBIHandler(
	incidents postgres.IncidentCauseRepository,
	predictions postgres.PredictionRepository,
	norms postgres.ConsumptionNormRepository,
	deviations consumption.Service,
	towers postgres.TowerRepository,
) *PowerBIHandler {
	return &PowerBIHandler{
		incidents:   incidents,
		predictions: predictions,
		norms:       norms,
		deviations:  deviations,
		towers:      towers,
	}
}

// Catalog — GET /api/v1/powerbi/catalog
// Lista os datasets disponíveis (útil como documentação dentro do Power BI).
func (h *PowerBIHandler) Catalog(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, []map[string]string{
		{"name": "incidents", "path": "/api/v1/powerbi/incidents?from=YYYY-MM-DD&to=YYYY-MM-DD", "description": "Incidentes de queda de site no período"},
		{"name": "kpis", "path": "/api/v1/powerbi/kpis?from=YYYY-MM-DD&to=YYYY-MM-DD", "description": "KPIs agregados (cartões)"},
		{"name": "consumption", "path": "/api/v1/powerbi/consumption?from=YYYY-MM-DD&to=YYYY-MM-DD", "description": "Desvios de consumo vs normas do Controller"},
		{"name": "predictions", "path": "/api/v1/powerbi/predictions", "description": "Previsão IA mais recente por torre"},
		{"name": "towers", "path": "/api/v1/powerbi/towers", "description": "Cache local de torres (vendor, disponibilidade)"},
	})
}

// Incidents — GET /api/v1/powerbi/incidents?from=YYYY-MM-DD&to=YYYY-MM-DD
func (h *PowerBIHandler) Incidents(w http.ResponseWriter, r *http.Request) {
	from, to, err := parsePeriod(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	list, err := h.incidents.ListByPeriod(r.Context(), from, to)
	if err != nil {
		http.Error(w, "erro ao listar incidentes", http.StatusInternalServerError)
		return
	}

	type row struct {
		ID                string   `json:"id"`
		TowerID           string   `json:"tower_id"`
		IncidentStartedAt string   `json:"incident_started_at"`
		IncidentEndedAt   *string  `json:"incident_ended_at,omitempty"`
		MLPredictedCause  *string  `json:"ml_predicted_cause,omitempty"`
		MLConfidence      *float64 `json:"ml_confidence,omitempty"`
		ConfirmedCause    *string  `json:"confirmed_cause,omitempty"`
		ConfirmedBy       *string  `json:"confirmed_by,omitempty"`
		ConfirmedAt       *string  `json:"confirmed_at,omitempty"`
		Status            string   `json:"status"`
		CreatedAt         string   `json:"created_at"`
	}

	out := make([]row, 0, len(list))
	for _, c := range list {
		item := row{
			ID:                c.ID,
			TowerID:           c.TowerID,
			IncidentStartedAt: c.IncidentStartedAt.UTC().Format(time.RFC3339),
			Status:            c.Status,
			CreatedAt:         c.CreatedAt.UTC().Format(time.RFC3339),
		}
		if c.IncidentEndedAt.Valid {
			s := c.IncidentEndedAt.Time.UTC().Format(time.RFC3339)
			item.IncidentEndedAt = &s
		}
		if c.MLPredictedCause.Valid {
			item.MLPredictedCause = &c.MLPredictedCause.String
		}
		if c.MLConfidence.Valid {
			item.MLConfidence = &c.MLConfidence.Float64
		}
		if c.ConfirmedCause.Valid {
			item.ConfirmedCause = &c.ConfirmedCause.String
		}
		if c.ConfirmedBy.Valid {
			item.ConfirmedBy = &c.ConfirmedBy.String
		}
		if c.ConfirmedAt.Valid {
			s := c.ConfirmedAt.Time.UTC().Format(time.RFC3339)
			item.ConfirmedAt = &s
		}
		out = append(out, item)
	}

	writeJSON(w, http.StatusOK, out)
}

// KPIs — GET /api/v1/powerbi/kpis?from=&to=
func (h *PowerBIHandler) KPIs(w http.ResponseWriter, r *http.Request) {
	from, to, err := parsePeriod(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	list, err := h.incidents.ListByPeriod(r.Context(), from, to)
	if err != nil {
		http.Error(w, "erro ao listar incidentes", http.StatusInternalServerError)
		return
	}

	sites := map[string]struct{}{}
	pending, confirmed, corrected := 0, 0, 0
	for _, c := range list {
		sites[c.TowerID] = struct{}{}
		switch c.Status {
		case "pending_confirmation":
			pending++
		case "confirmed":
			confirmed++
		case "corrected":
			corrected++
		}
	}

	// Contagem de desvios fora da norma (best-effort).
	outOfNorm := 0
	if h.norms != nil && h.deviations != nil {
		if norms, err := h.norms.ListActive(r.Context()); err == nil {
			seen := map[string]struct{}{}
			for _, n := range norms {
				if _, ok := seen[n.TowerID]; ok {
					continue
				}
				seen[n.TowerID] = struct{}{}
				devs, err := h.deviations.ComputeForTower(r.Context(), n.TowerID, from, to)
				if err != nil {
					continue
				}
				for _, d := range devs {
					if !d.WithinNorm {
						outOfNorm++
					}
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"period_from":          from.Format("2006-01-02"),
		"period_to":            to.Format("2006-01-02"),
		"total_incidents":      len(list),
		"sites_affected":       len(sites),
		"pending_confirmation": pending,
		"confirmed":            confirmed,
		"corrected":            corrected,
		"consumption_out_of_norm": outOfNorm,
		"generated_at":         time.Now().UTC().Format(time.RFC3339),
	})
}

// Consumption — GET /api/v1/powerbi/consumption?from=&to=
// Desvios de consumo vs normas do Controller — dataset chave para Financeiro
// e Controller no Power BI.
func (h *PowerBIHandler) Consumption(w http.ResponseWriter, r *http.Request) {
	if h.norms == nil || h.deviations == nil {
		http.Error(w, "serviço de consumo não configurado", http.StatusServiceUnavailable)
		return
	}

	from, to, err := parsePeriod(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	norms, err := h.norms.ListActive(r.Context())
	if err != nil {
		http.Error(w, "erro ao listar normas", http.StatusInternalServerError)
		return
	}

	type row struct {
		TowerID          string  `json:"tower_id"`
		EquipmentType    string  `json:"equipment_type"`
		ExpectedValue    float64 `json:"expected_value"`
		AverageActual    float64 `json:"average_actual"`
		Unit             string  `json:"unit"`
		TolerancePercent float64 `json:"tolerance_percent"`
		DeviationPercent float64 `json:"deviation_percent"`
		WithinNorm       bool    `json:"within_norm"`
		SampleCount      int     `json:"sample_count"`
		PeriodFrom       string  `json:"period_from"`
		PeriodTo         string  `json:"period_to"`
	}

	seen := map[string]struct{}{}
	out := make([]row, 0)
	for _, n := range norms {
		if _, ok := seen[n.TowerID]; ok {
			continue
		}
		seen[n.TowerID] = struct{}{}

		devs, err := h.deviations.ComputeForTower(r.Context(), n.TowerID, from, to)
		if err != nil {
			continue
		}
		for _, d := range devs {
			out = append(out, row{
				TowerID:          d.TowerID,
				EquipmentType:    d.EquipmentType,
				ExpectedValue:    d.ExpectedValue,
				AverageActual:    d.AverageActual,
				Unit:             d.Unit,
				TolerancePercent: d.TolerancePercent,
				DeviationPercent: d.DeviationPercent,
				WithinNorm:       d.WithinNorm,
				SampleCount:      d.SampleCount,
				PeriodFrom:       from.Format("2006-01-02"),
				PeriodTo:         to.Format("2006-01-02"),
			})
		}
	}

	writeJSON(w, http.StatusOK, out)
}

// Predictions — GET /api/v1/powerbi/predictions
// Previsão IA mais recente por torre (sem filtro de data — snapshot atual).
func (h *PowerBIHandler) Predictions(w http.ResponseWriter, r *http.Request) {
	if h.predictions == nil {
		http.Error(w, "repositório de previsões não configurado", http.StatusServiceUnavailable)
		return
	}

	list, err := h.predictions.ListLatestPerTower(r.Context(), 500)
	if err != nil {
		http.Error(w, "erro ao listar previsões", http.StatusInternalServerError)
		return
	}

	type row struct {
		TowerID          string  `json:"tower_id"`
		Model            string  `json:"model"`
		Score            float64 `json:"score"`
		Status           string  `json:"status"`
		Explanation      string  `json:"explanation"`
		PredictionWindow int     `json:"prediction_window_days"`
		CreatedAt        string  `json:"created_at"`
	}

	out := make([]row, 0, len(list))
	for _, p := range list {
		out = append(out, row{
			TowerID:          p.TowerID,
			Model:            p.Model,
			Score:            p.Score,
			Status:           p.Status,
			Explanation:      p.Explanation,
			PredictionWindow: p.PredictionWindow,
			CreatedAt:        p.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, out)
}

// Towers — GET /api/v1/powerbi/towers
// Cache local de torres (vendor, disponibilidade) — dimensão para joins no Power BI.
func (h *PowerBIHandler) Towers(w http.ResponseWriter, r *http.Request) {
	if h.towers == nil {
		http.Error(w, "repositório de torres não configurado", http.StatusServiceUnavailable)
		return
	}

	list, err := h.towers.ListAll(r.Context())
	if err != nil {
		http.Error(w, "erro ao listar torres", http.StatusInternalServerError)
		return
	}

	type row struct {
		TowerID         string   `json:"tower_id"`
		Name            string   `json:"name"`
		Vendor          string   `json:"vendor"`
		OperatorID      string   `json:"operator_id"`
		RegionID        string   `json:"region_id"`
		Availability7d  *float64 `json:"availability_7d,omitempty"`
		Availability30d *float64 `json:"availability_30d,omitempty"`
	}

	out := make([]row, 0, len(list))
	for _, t := range list {
		item := row{
			TowerID:    t.TowerID,
			Name:       t.Name,
			Vendor:     t.Vendor,
			OperatorID: t.OperatorID,
			RegionID:   t.RegionID,
		}
		if t.Availability7d.Valid {
			v := t.Availability7d.Float64
			item.Availability7d = &v
		}
		if t.Availability30d.Valid {
			v := t.Availability30d.Float64
			item.Availability30d = &v
		}
		out = append(out, item)
	}

	writeJSON(w, http.StatusOK, out)
}
