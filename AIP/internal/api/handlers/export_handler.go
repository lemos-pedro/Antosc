package handlers

import (
	"net/http"
	"time"

	"github.com/antosc/aip/internal/reports"
	"github.com/antosc/aip/internal/repository/postgres"
	"github.com/antosc/aip/internal/services/consumption"
)

type ExportHandler struct {
	incidents  postgres.IncidentCauseRepository
	deviations consumption.Service
}

func NewExportHandler(incidents postgres.IncidentCauseRepository, deviations consumption.Service) *ExportHandler {
	return &ExportHandler{incidents: incidents, deviations: deviations}
}

// IncidentsExport — GET /api/v1/reports/export/incidents?from=&to=
// Excel com os sites caídos no período + causa (ML e confirmada pelo O&M).
func (h *ExportHandler) IncidentsExport(w http.ResponseWriter, r *http.Request) {
	from, to, err := parsePeriod(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	list, err := h.incidents.ListByPeriod(r.Context(), from, to)
	if err != nil {
		http.Error(w, "erro ao obter incidentes", http.StatusInternalServerError)
		return
	}

	rows := make([][]any, 0, len(list))
	for _, c := range list {
		cause := ""
		if c.MLPredictedCause.Valid {
			cause = c.MLPredictedCause.String
		}
		confirmed := ""
		if c.ConfirmedCause.Valid {
			confirmed = c.ConfirmedCause.String
		}
		rows = append(rows, []any{c.TowerID, c.IncidentStartedAt.Format("2006-01-02 15:04"), cause, confirmed, c.Status})
	}

	xlsx, err := reports.BuildExcel([]reports.Table{{
		SheetName: "Sites Caidos",
		Headers:   []string{"Tower ID", "Início do Incidente", "Causa (ML)", "Causa Confirmada", "Estado"},
		Rows:      rows,
	}})
	if err != nil {
		http.Error(w, "erro ao gerar Excel", http.StatusInternalServerError)
		return
	}

	writeXLSX(w, "sites_caidos.xlsx", xlsx)
}

// ConsumptionExport — GET /api/v1/reports/export/consumption/{tower_id}?from=&to=
func (h *ExportHandler) ConsumptionExport(w http.ResponseWriter, r *http.Request) {
	towerID := r.PathValue("tower_id")

	from, to, err := parsePeriod(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	devs, err := h.deviations.ComputeForTower(r.Context(), towerID, from, to)
	if err != nil {
		http.Error(w, "erro ao calcular desvios", http.StatusInternalServerError)
		return
	}

	rows := make([][]any, 0, len(devs))
	for _, d := range devs {
		rows = append(rows, []any{d.EquipmentType, d.ExpectedValue, d.AverageActual, d.Unit, d.DeviationPercent, d.WithinNorm, d.SampleCount})
	}

	xlsx, err := reports.BuildExcel([]reports.Table{{
		SheetName: "Consumo",
		Headers:   []string{"Equipamento", "Valor Esperado", "Média Real", "Unidade", "Desvio %", "Dentro da Norma", "Nº Amostras"},
		Rows:      rows,
	}})
	if err != nil {
		http.Error(w, "erro ao gerar Excel", http.StatusInternalServerError)
		return
	}

	writeXLSX(w, "consumo_"+towerID+".xlsx", xlsx)
}

func parsePeriod(r *http.Request) (from, to time.Time, err error) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	from, err = time.Parse("2006-01-02", fromStr)
	if err != nil {
		return time.Time{}, time.Time{}, errInvalidFrom
	}
	to, err = time.Parse("2006-01-02", toStr)
	if err != nil {
		to = from.AddDate(0, 0, 7)
	}
	return from, to, nil
}

var errInvalidFrom = httpErr("parâmetro 'from' inválido, use YYYY-MM-DD")

type httpErr string

func (e httpErr) Error() string { return string(e) }

func writeXLSX(w http.ResponseWriter, filename string, data []byte) {
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
