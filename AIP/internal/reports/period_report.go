package reports

import (
	"bytes"
	"fmt"
	"strings"
	"html/template"
	"time"

	"github.com/antosc/aip/internal/repository/postgres"
	"github.com/antosc/aip/internal/services/consumption"
)

// PeriodKind distingue relatório semanal de mensal (e futuros trimestrais).
type PeriodKind string

const (
	PeriodWeekly  PeriodKind = "weekly"
	PeriodMonthly PeriodKind = "monthly"
)

// PeriodReport é o dataset unificado que alimenta todos os templates por persona.
// Mantém a regra "no fabricated data": só contém o que veio dos repositórios.
type PeriodReport struct {
	Kind       PeriodKind
	PeriodFrom time.Time
	PeriodTo   time.Time
	Generated  time.Time

	Incidents  []postgres.IncidentCause
	// Deviations agregadas (todas as torres com norma ativa) — preenchido
	// quando o job tem acesso ao serviço de consumo. Pode ficar vazio.
	Deviations []consumption.Deviation

	// RiskSites: previsões IA ordenadas por risco (score ascendente).
	// Vazio se predict_batch ainda não correu.
	RiskSites []RiskSite

	// KPIs pré-calculados (derivados, não inventados).
	TotalIncidents      int
	PendingConfirmation int
	Confirmed           int
	Corrected           int
	SitesAffected       int
	AtRiskCount         int // status critical/at_risk/anomaly/warning

	// Risk portfolio (Monte Carlo) — preenchido pelo report_job se houver previsões.
	RiskHorizonDays      int
	ExpectedFailures     float64
	P50Failures          float64
	P90Failures          float64
	ProbAtLeastOneFail   float64
}

// RiskSite é uma linha do ranking de risco para HTML/Excel.
type RiskSite struct {
	TowerID     string
	Model       string
	Score       float64
	Status      string
	Explanation string
	CreatedAt   time.Time
}

// BuildPeriodReport monta o dataset a partir dos dados já obtidos.
// risk pode ser nil/vazio se o predict_batch ainda não populou ai_predictions.
func BuildPeriodReport(kind PeriodKind, from, to time.Time, incidents []postgres.IncidentCause, deviations []consumption.Deviation, risk []RiskSite) PeriodReport {
	sites := map[string]struct{}{}
	pending, confirmed, corrected := 0, 0, 0
	for _, c := range incidents {
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

	atRisk := 0
	for _, s := range risk {
		switch s.Status {
		case "critical", "critical_now", "at_risk", "anomaly", "warning":
			atRisk++
		}
	}

	return PeriodReport{
		Kind:                kind,
		PeriodFrom:          from,
		PeriodTo:            to,
		Generated:           time.Now(),
		Incidents:           incidents,
		Deviations:          deviations,
		RiskSites:           risk,
		TotalIncidents:      len(incidents),
		PendingConfirmation: pending,
		Confirmed:           confirmed,
		Corrected:           corrected,
		SitesAffected:       len(sites),
		AtRiskCount:         atRisk,
	}
}

// Subject linha de assunto do email.
func (r PeriodReport) Subject(meta PersonaMeta) string {
	label := "Semanal"
	if r.Kind == PeriodMonthly {
		label = "Mensal"
	}
	return fmt.Sprintf("AIP — Relatório %s · %s (%s a %s)",
		label, meta.DisplayName,
		r.PeriodFrom.Format("02/01"), r.PeriodTo.Format("02/01"))
}

// Filename do anexo Excel.
func (r PeriodReport) Filename(meta PersonaMeta) string {
	kind := "semanal"
	if r.Kind == PeriodMonthly {
		kind = "mensal"
	}
	return fmt.Sprintf("aip_%s_%s_%s.xlsx", kind, meta.ID, r.PeriodTo.Format("2006-01-02"))
}

// --- HTML templates por família de persona ---

type htmlView struct {
	Meta    PersonaMeta
	Report  PeriodReport
	// Incidents limitados para o email (não para o Excel).
	Incidents []postgres.IncidentCause
	// Texto introdutório adaptado.
	Intro string
	// Resumo executivo (CEO/Conselho) — opcional, gerado via Ollama ou fallback.
	ExecutiveSummary string
	// Secção de ações (O&M / Engenharia / Diretor Técnico).
	Actions []string
	// Ranking de risco (previsões).
	RiskSites []RiskSite
	// Nota interativa (assistente).
	InteractiveNote string
}

const periodHTMLTmpl = `
<!DOCTYPE html>
<html lang="pt">
<head><meta charset="UTF-8"><title>{{.Meta.DisplayName}}</title></head>
<body style="font-family:Segoe UI,Arial,sans-serif;font-size:14px;color:#222;max-width:720px;margin:0 auto;padding:16px">
  <h2 style="margin-bottom:4px">Relatório {{if eq .Report.Kind "monthly"}}Mensal{{else}}Semanal{{end}} AIP — {{.Meta.DisplayName}}</h2>
  <p style="color:#666;margin-top:0">{{.Report.PeriodFrom.Format "02/01/2006"}} a {{.Report.PeriodTo.Format "02/01/2006"}} · Gerado {{.Report.Generated.Format "02/01 15:04"}}</p>

  <p><strong>Foco:</strong> {{.Meta.Focus}}</p>
  <p>{{.Intro}}</p>

  {{if .ExecutiveSummary}}
  <div style="background:#eef4ff;border-left:4px solid #2b6cb0;padding:12px 16px;margin:16px 0">
    <strong style="color:#2b6cb0">Resumo executivo</strong>
    <p style="margin:8px 0 0 0">{{.ExecutiveSummary}}</p>
  </div>
  {{end}}

  <table style="border-collapse:collapse;margin:16px 0;width:100%">
    <tr>
      <td style="background:#f5f5f5;padding:10px;border:1px solid #ddd;text-align:center"><strong>{{.Report.TotalIncidents}}</strong><br><span style="font-size:12px;color:#666">Incidentes</span></td>
      <td style="background:#f5f5f5;padding:10px;border:1px solid #ddd;text-align:center"><strong>{{.Report.SitesAffected}}</strong><br><span style="font-size:12px;color:#666">Sites afetados</span></td>
      <td style="background:#fff3cd;padding:10px;border:1px solid #ddd;text-align:center"><strong>{{.Report.PendingConfirmation}}</strong><br><span style="font-size:12px;color:#666">Pendentes O&amp;M</span></td>
      <td style="background:#d4edda;padding:10px;border:1px solid #ddd;text-align:center"><strong>{{.Report.Confirmed}}</strong><br><span style="font-size:12px;color:#666">Confirmados</span></td>
      <td style="background:#fce8e6;padding:10px;border:1px solid #ddd;text-align:center"><strong>{{.Report.AtRiskCount}}</strong><br><span style="font-size:12px;color:#666">Em risco (IA)</span></td>
      {{if gt .Report.RiskHorizonDays 0}}
      <td style="background:#fff3cd;padding:10px;border:1px solid #ddd;text-align:center"><strong>{{printf "%.1f" .Report.ExpectedFailures}}</strong><br><span style="font-size:12px;color:#666">Falhas esperadas {{.Report.RiskHorizonDays}}d</span></td>
      <td style="background:#fff3cd;padding:10px;border:1px solid #ddd;text-align:center"><strong>{{printf "%.0f" .Report.P90Failures}}</strong><br><span style="font-size:12px;color:#666">P90 falhas {{.Report.RiskHorizonDays}}d</span></td>
      {{end}}
    </tr>
  </table>

  {{if .Actions}}
  <h3 style="margin-bottom:8px">Ações prioritárias</h3>
  <ol style="padding-left:20px;margin-top:0">
    {{range .Actions}}<li style="margin-bottom:6px">{{.}}</li>{{end}}
  </ol>
  {{end}}

  {{if .RiskSites}}
  <h3 style="margin-bottom:8px">Ranking de risco (previsão IA)</h3>
  <p style="font-size:12px;color:#666;margin-top:0">{{.Report.AtRiskCount}} site(s) em estado de atenção no snapshot atual.</p>
  <table border="1" cellpadding="6" cellspacing="0" style="border-collapse:collapse;font-size:13px;width:100%">
    <tr style="background:#fce8e6"><th>Site</th><th>Score</th><th>Estado</th><th>Modelo</th><th>Explicação</th></tr>
    {{range .RiskSites}}
    <tr>
      <td>{{.TowerID}}</td>
      <td>{{printf "%.1f" .Score}}</td>
      <td>{{.Status}}</td>
      <td>{{.Model}}</td>
      <td>{{.Explanation}}</td>
    </tr>
    {{end}}
  </table>
  {{end}}

  {{if .Incidents}}
  <h3 style="margin-bottom:8px">Incidentes no período</h3>
  <table border="1" cellpadding="6" cellspacing="0" style="border-collapse:collapse;font-size:13px;width:100%">
    <tr style="background:#f0f0f0">
      <th>Site</th><th>Início</th><th>Causa (ML)</th><th>Confirmada</th><th>Estado</th>
    </tr>
    {{range .Incidents}}
    <tr>
      <td>{{.TowerID}}</td>
      <td>{{.IncidentStartedAt.Format "02/01 15:04"}}</td>
      <td>{{if .MLPredictedCause.Valid}}{{.MLPredictedCause.String}}{{else}}—{{end}}</td>
      <td>{{if .ConfirmedCause.Valid}}{{.ConfirmedCause.String}}{{else}}pendente{{end}}</td>
      <td>{{.Status}}</td>
    </tr>
    {{end}}
  </table>
  {{if gt (len .Report.Incidents) (len .Incidents)}}
  <p style="font-size:12px;color:#888">A mostrar {{len .Incidents}} de {{len .Report.Incidents}} — detalhe completo no Excel anexo.</p>
  {{end}}
  {{end}}

  <hr style="border:none;border-top:1px solid #eee;margin:24px 0">
  <p style="font-size:12px;color:#555">{{.InteractiveNote}}</p>
  <p style="font-size:11px;color:#999">Excel em anexo. Gerado automaticamente pelo AIP · Antosc Intelligence Platform.</p>
</body>
</html>
`

var periodTmpl = template.Must(template.New("period").Parse(periodHTMLTmpl))

// BuildPersonaHTML gera o corpo HTML do email adaptado à persona.
// executiveSummary é opcional (vazio para personas técnicas).
func BuildPersonaHTML(r PeriodReport, meta PersonaMeta, executiveSummary string) (string, error) {
	incidents := r.Incidents
	if meta.MaxIncidentsInHTML > 0 && len(incidents) > meta.MaxIncidentsInHTML {
		incidents = incidents[:meta.MaxIncidentsInHTML]
	}

	risk := []RiskSite(nil)
	if meta.IncludeRisk && len(r.RiskSites) > 0 {
		risk = r.RiskSites
		if meta.MaxRiskInHTML > 0 && len(risk) > meta.MaxRiskInHTML {
			risk = risk[:meta.MaxRiskInHTML]
		}
	}

	view := htmlView{
		Meta:             meta,
		Report:           r,
		Incidents:        incidents,
		Intro:            introFor(meta, r),
		ExecutiveSummary: executiveSummary,
		Actions:          actionsFor(meta, r),
		RiskSites:        risk,
		InteractiveNote:  interactiveNote(meta),
	}

	var buf bytes.Buffer
	if err := periodTmpl.Execute(&buf, view); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func introFor(meta PersonaMeta, r PeriodReport) string {
	switch meta.ID {
	case PersonaCEO, PersonaConselho:
		if r.TotalIncidents == 0 {
			return "No período não foram registados incidentes de queda de site. A rede manteve-se estável."
		}
		return fmt.Sprintf(
			"Foram registados %d incidente(s) em %d site(s). %d aguardam confirmação do O&M. O detalhe operacional está no anexo e pode ser explorado via assistente AIP.",
			r.TotalIncidents, r.SitesAffected, r.PendingConfirmation,
		)
	case PersonaFinanceiro:
		return "Resumo orientado a custo e conformidade de consumo. Use o Excel e o assistente (role=financeiro) para aprofundar desvios."
	case PersonaController:
		return "Foco em conformidade face às normas que definiu. Sites fora da tolerância e incidentes pendentes de confirmação."
	case PersonaOM:
		return "Priorize os sites com estado pending_confirmation e as causas críticas. Cada linha do Excel tem a causa prevista pelo ML para validar ou corrigir."
	case PersonaEngenharia:
		return "Análise técnica do período: causas ML vs confirmadas (sinal de retreino) e padrões recorrentes."
	default:
		return "Resumo do período com incidentes, estado de confirmação e anexo detalhado."
	}
}

func actionsFor(meta PersonaMeta, r PeriodReport) []string {
	if !meta.IncludeActions {
		return nil
	}
	var out []string
	if r.PendingConfirmation > 0 {
		out = append(out, fmt.Sprintf("Confirmar ou corrigir %d causa(s) pendente(s) em POST /api/v1/incidents/{id}/confirm — alimenta o retreino do modelo.", r.PendingConfirmation))
	}
	if r.AtRiskCount > 0 {
		out = append(out, fmt.Sprintf("Rever %d site(s) em risco no ranking de previsão IA (folha Ranking Risco no Excel).", r.AtRiskCount))
	}
	if r.ExpectedFailures >= 1 {
		out = append(out, fmt.Sprintf("Monte Carlo (%dd): ~%.1f falhas esperadas (P90=%.0f). Priorizar sites critical/at_risk.", r.RiskHorizonDays, r.ExpectedFailures, r.P90Failures))
	}
	// Prescrições embutidas nas explanations do ranking
	for _, s := range r.RiskSites {
		if len(out) >= 8 {
			break
		}
		if strings.Contains(s.Explanation, "Prescrições:") || strings.Contains(s.Explanation, "substituir bateria") {
			out = append(out, fmt.Sprintf("%s: %s", s.TowerID, trimExpl(s.Explanation, 160)))
		}
	}
	if r.TotalIncidents > 0 {
		out = append(out, "Rever sites com severidade crítica / power_loss no Excel e planear deslocações se necessário.")
	}
	if meta.ID == PersonaDiretorTec && r.SitesAffected > 3 {
		out = append(out, "Avaliar padrão cross-site: se vários sites partilham vendor/região, priorizar intervenção de engenharia.")
	}
	if len(out) == 0 {
		out = append(out, "Nenhuma ação urgente identificada neste período — manter monitorização habitual.")
	}
	return out
}

func interactiveNote(meta PersonaMeta) string {
	return fmt.Sprintf(
		`Perguntas interativas: use o assistente AIP (POST /api/v1/assistant/ask) com "role": "%s". Exemplos: "quais sites caíram esta semana?", "desvios de consumo do site X", "prioridades para amanhã". As respostas são grounded nos dados reais da API.`,
		meta.ID,
	)
}

// BuildPersonaExcel gera o .xlsx adaptado à persona.
func BuildPersonaExcel(r PeriodReport, meta PersonaMeta) ([]byte, error) {
	tables := []Table{}

	// Folha 1 — sempre: resumo KPI
	tables = append(tables, Table{
		SheetName: "Resumo",
		Headers:   []string{"Métrica", "Valor"},
		Rows: [][]any{
			{"Período de", r.PeriodFrom.Format("2006-01-02")},
			{"Período até", r.PeriodTo.Format("2006-01-02")},
			{"Tipo", string(r.Kind)},
			{"Persona", meta.DisplayName},
			{"Total incidentes", r.TotalIncidents},
			{"Sites afetados", r.SitesAffected},
			{"Pendentes confirmação", r.PendingConfirmation},
			{"Confirmados", r.Confirmed},
			{"Corrigidos (ML ≠ O&M)", r.Corrected},
			{"Sites em risco (IA)", r.AtRiskCount},
			{"MC horizonte (dias)", r.RiskHorizonDays},
			{"MC falhas esperadas", r.ExpectedFailures},
			{"MC P50 falhas", r.P50Failures},
			{"MC P90 falhas", r.P90Failures},
			{"MC P(≥1 falha)", r.ProbAtLeastOneFail},
		},
	})

	// Folha 2 — incidentes (detalhe completo se IncludeDetail, senão top N)
	incRows := make([][]any, 0, len(r.Incidents))
	limit := len(r.Incidents)
	if !meta.IncludeDetail && meta.MaxIncidentsInHTML > 0 && limit > meta.MaxIncidentsInHTML {
		limit = meta.MaxIncidentsInHTML
	}
	for i := 0; i < limit; i++ {
		c := r.Incidents[i]
		cause, confirmed := "", "pendente"
		if c.MLPredictedCause.Valid {
			cause = c.MLPredictedCause.String
		}
		if c.ConfirmedCause.Valid {
			confirmed = c.ConfirmedCause.String
		}
		incRows = append(incRows, []any{
			c.TowerID,
			c.IncidentStartedAt.Format("2006-01-02 15:04"),
			cause,
			confirmed,
			c.Status,
		})
	}
	tables = append(tables, Table{
		SheetName: "Incidentes",
		Headers:   []string{"Tower ID", "Início", "Causa (ML)", "Causa Confirmada", "Estado"},
		Rows:      incRows,
	})

	// Folha — ranking de risco
	if meta.IncludeRisk && len(r.RiskSites) > 0 {
		riskRows := make([][]any, 0, len(r.RiskSites))
		for _, s := range r.RiskSites {
			riskRows = append(riskRows, []any{s.TowerID, s.Score, s.Status, s.Model, s.Explanation, s.CreatedAt.Format("2006-01-02 15:04")})
		}
		tables = append(tables, Table{
			SheetName: "Ranking Risco",
			Headers:   []string{"Tower ID", "Score", "Estado", "Modelo", "Explicação", "Previsto em"},
			Rows:      riskRows,
		})
	}

	// Folha — desvios de consumo (só personas financeiras / controller / diretor)
	if meta.IncludeFinancial && len(r.Deviations) > 0 {
		devRows := make([][]any, 0, len(r.Deviations))
		for _, d := range r.Deviations {
			within := "Sim"
			if !d.WithinNorm {
				within = "NÃO"
			}
			devRows = append(devRows, []any{
				d.TowerID, d.EquipmentType, d.ExpectedValue, d.AverageActual,
				d.Unit, d.DeviationPercent, within, d.SampleCount,
			})
		}
		tables = append(tables, Table{
			SheetName: "Consumo vs Norma",
			Headers: []string{
				"Tower ID", "Equipamento", "Esperado", "Média real",
				"Unidade", "Desvio %", "Dentro norma", "Amostras",
			},
			Rows: devRows,
		})
	}

	return BuildExcel(tables)
}


func trimExpl(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
