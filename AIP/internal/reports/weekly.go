package reports

import (
	"bytes"
	"fmt"
	"html/template"
	"time"

	"github.com/antosc/aip/internal/repository/postgres"
)

// WeeklyReport agrega os dados que entram no relatório semanal.
type WeeklyReport struct {
	PeriodFrom time.Time
	PeriodTo   time.Time
	Incidents  []postgres.IncidentCause
}

const weeklyHTMLTemplate = `
<h2>Relatório Semanal AIP — {{.PeriodFrom.Format "02/01/2006"}} a {{.PeriodTo.Format "02/01/2006"}}</h2>
<p>{{len .Incidents}} site(s) com incidente registado nesta semana.</p>
<table border="1" cellpadding="6" cellspacing="0" style="border-collapse:collapse;font-family:sans-serif;font-size:13px">
<tr style="background:#f0f0f0"><th>Site</th><th>Início</th><th>Causa (ML)</th><th>Causa Confirmada</th><th>Estado</th></tr>
{{range .Incidents}}
<tr>
<td>{{.TowerID}}</td>
<td>{{.IncidentStartedAt.Format "02/01 15:04"}}</td>
<td>{{if .MLPredictedCause.Valid}}{{.MLPredictedCause.String}}{{else}}--{{end}}</td>
<td>{{if .ConfirmedCause.Valid}}{{.ConfirmedCause.String}}{{else}}pendente{{end}}</td>
<td>{{.Status}}</td>
</tr>
{{end}}
</table>
<p style="color:#888;font-size:12px">Excel em anexo com o detalhe completo. Gerado automaticamente pelo AIP.</p>
`

var weeklyTmpl = template.Must(template.New("weekly").Parse(weeklyHTMLTemplate))

// BuildWeeklyHTML gera o corpo HTML do email do relatório semanal.
func BuildWeeklyHTML(r WeeklyReport) (string, error) {
	var buf bytes.Buffer
	if err := weeklyTmpl.Execute(&buf, r); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// BuildWeeklyExcel gera o anexo .xlsx com o mesmo conteúdo do relatório.
func BuildWeeklyExcel(r WeeklyReport) ([]byte, error) {
	rows := make([][]any, 0, len(r.Incidents))
	for _, c := range r.Incidents {
		cause, confirmed := "", "pendente"
		if c.MLPredictedCause.Valid {
			cause = c.MLPredictedCause.String
		}
		if c.ConfirmedCause.Valid {
			confirmed = c.ConfirmedCause.String
		}
		rows = append(rows, []any{c.TowerID, c.IncidentStartedAt.Format("2006-01-02 15:04"), cause, confirmed, c.Status})
	}

	return BuildExcel([]Table{{
		SheetName: "Relatorio Semanal",
		Headers:   []string{"Tower ID", "Início do Incidente", "Causa (ML)", "Causa Confirmada", "Estado"},
		Rows:      rows,
	}})
}

func WeeklySubject(r WeeklyReport) string {
	return fmt.Sprintf("AIP — Relatório Semanal (%s a %s)", r.PeriodFrom.Format("02/01"), r.PeriodTo.Format("02/01"))
}
