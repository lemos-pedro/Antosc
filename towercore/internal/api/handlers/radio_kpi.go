// Package handlers contém implementations de handlers HTTP para varios endpoints.
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"towercore/internal/core/domain"
	"towercore/internal/core/services"
)

// RadioKPIHandler lida com requisições HTTP relacionadas a KPIs de rádio federados.
type RadioKPIHandler struct {
	service *services.RadioKPIService
}

// NewRadioKPIHandler cria um novo handler para KPIs de rádio.
func NewRadioKPIService(service *services.RadioKPIService) *RadioKPIHandler {
	return &RadioKPIHandler{service: service}
}

// ServeHTTP implementa a interface http.Handler.
func (h *RadioKPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.ingestKPIs(w, r)
	case http.MethodGet:
		h.getKPIs(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// ingestKPIs lida com POST /api/radio-kpi para receber KPIs de rádio federados.
// Espera um JSON array de objetos RadioKPI no corpo da requisição.
func (h *RadioKPIHandler) ingestKPIs(w http.ResponseWriter, r *http.Request) {
	var kpis []*domain.RadioKPI
	if err := json.NewDecoder(r.Body).Decode(&kpis); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.service.IngestKPIs(r.Context(), kpis); err != nil {
		http.Error(w, "failed to ingest radio KPIs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"count":   strconv.Itoa(len(kpis)),
		"message": fmt.Sprintf("ingested %d radio KPIs", len(kpis)),
	})
}

// getKPIs lida com GET /api/radio-kpi para consultar KPIs de rádio.
// Suporta query parameters para filtragem:
//   - tower_id: UUID da torre
//   - sector_id: ID do setor (opcional)
//   - technique: tecnologia de celular (LTE, 5GNR, etc.) (opcional)
//   - limit: número máximo de resultados (padrão: 100)
//   - offset: offset para paginação (padrão: 0)
//   - order_by: campo para ordenação (padrão: measured_at desc)
//   - from: timestamp inicial (ISO 8601)
//   - to: timestamp final (ISO 8601)
func (h *RadioKPIHandler) getKPIs(w http.ResponseWriter, r *http.Request) {
	// Parse da torre_id (obrigatório)
	towerIDStr := r.URL.Query().Get("tower_id")
	if towerIDStr == "" {
		http.Error(w, "tower_id query parameter is required", http.StatusBadRequest)
		return
	}

	towerID, err := uuid.Parse(towerIDStr)
	if err != nil {
		http.Error(w, "invalid tower_id format", http.StatusBadRequest)
		return
	}

	// Parse dos parâmetros opcionais
	sectorIDStr := r.URL.Query().Get("sector_id")
	var sectorID *string
	if sectorIDStr != "" {
		sectorID = &sectorIDStr
	}

	techniqueStr := r.URL.Query().Get("technique")
	var technique *string
	if techniqueStr != "" {
		technique = &techniqueStr
	}

	// Parse de limite e offset
	limit := 100 // padrão
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l >= 0 {
			limit = l
		}
	}

	offset := 0 // padrão
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Parse de range de datas (opcional)
	var measuredAfter, measuredBefore time.Time
	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			measuredAfter = t
		}
	}
	if toStr := r.URL.Query().Get("to"); toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			measuredBefore = t
		}
	}

	// Construir o filtro
	filter := &domain.RadioKPIFilter{
		TowerID:         towerID,
		SectorID:        sectorID,
		CellTechnique:   technique,
		Limit:           limit,
		Offset:          offset,
		OrderBy:         []string{"measured_at DESC"},
		MeasuredAtAfter: &measuredAfter,
		MeasuredAtBefore: &measuredBefore,
	}

	// Buscar os KPIs
	kpis, total, err := h.service.GetKPIsHistorico(r.Context(), filter)
	if err != nil {
		http.Error(w, "failed to retrieve radio KPIs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Retornar resposta com metadados de paginação
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"count":   len(kpis),
		"total":   total,
		"limit":   limit,
		"offset":  offset,
		"data":    kpis,
	})
}