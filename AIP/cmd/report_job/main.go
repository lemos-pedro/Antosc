// cmd/report_job gera e envia relatórios semanais ou mensais por persona
// via Resend. Substitui o antigo weekly_report (mantido por compatibilidade).
//
// Uso:
//
//	REPORT_PERIOD=weekly|monthly go run ./cmd/report_job
//
// Variáveis de ambiente (além das do config habitual):
//
//	RESEND_API_KEY          obrigatório
//	RESEND_FROM             opcional (default alerts@antosc.com)
//	REPORT_PERIOD           weekly (default) | monthly
//	OLLAMA_URL / OLLAMA_MODEL  usados no resumo executivo (CEO/Conselho)
//	REPORT_RECIPIENTS_OM=...
//	REPORT_RECIPIENTS_ENGENHARIA=...
//	REPORT_RECIPIENTS_CONTROLLER=...
//	REPORT_RECIPIENTS_FINANCEIRO=...
//	REPORT_RECIPIENTS_DIRETOR_TECNICO=...
//	REPORT_RECIPIENTS_CEO=...
//	REPORT_RECIPIENTS_CONSELHO_ADMINISTRACAO=...
//
// Só envia para personas com destinatários definidos. Personas sem emails
// são omitidas sem erro.
package main

import (
	"context"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/antosc/aip/internal/analytics"
	"github.com/antosc/aip/internal/config"
	"github.com/antosc/aip/internal/logger"
	"github.com/antosc/aip/internal/notification"
	"github.com/antosc/aip/internal/reports"
	"github.com/antosc/aip/internal/repository/postgres"
	"github.com/antosc/aip/internal/services/consumption"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.LogLevel)

	if cfg.ResendAPIKey == "" {
		log.Error("RESEND_API_KEY não definido")
		os.Exit(1)
	}

	period := reports.PeriodWeekly
	if strings.EqualFold(os.Getenv("REPORT_PERIOD"), "monthly") {
		period = reports.PeriodMonthly
	}

	to := time.Now()
	var from time.Time
	if period == reports.PeriodMonthly {
		from = to.AddDate(0, -1, 0)
	} else {
		from = to.AddDate(0, 0, -7)
	}

	db, err := postgres.New(cfg.Database.DSN())
	if err != nil {
		log.Error("ligação à base de dados falhou", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	incidentRepo := postgres.NewIncidentCauseRepository(db)
	incidents, err := incidentRepo.ListByPeriod(ctx, from, to)
	if err != nil {
		log.Error("erro ao obter incidentes", "err", err)
		os.Exit(1)
	}

	// Desvios de consumo: tenta agregar por normas ativas (best-effort).
	var deviations []consumption.Deviation
	normRepo := postgres.NewConsumptionNormRepository(db)
	featureRepo := postgres.NewFeatureRepository(db)
	devService := consumption.NewService(normRepo, featureRepo)
	if norms, err := normRepo.ListActive(ctx); err == nil {
		seen := map[string]struct{}{}
		for _, n := range norms {
			if _, ok := seen[n.TowerID]; ok {
				continue
			}
			seen[n.TowerID] = struct{}{}
			devs, err := devService.ComputeForTower(ctx, n.TowerID, from, to)
			if err != nil {
				log.Warn("desvios de consumo falharam para torre", "tower_id", n.TowerID, "err", err)
				continue
			}
			deviations = append(deviations, devs...)
		}
	} else {
		log.Warn("não foi possível listar normas ativas — relatório sem folha de consumo", "err", err)
	}

	// Ranking de risco a partir das previsões mais recentes (predict_batch).
	var risk []reports.RiskSite
	var preds []postgres.Prediction
	predictionRepo := postgres.NewPredictionRepository(db)
	if list, err := predictionRepo.ListLatestPerTower(ctx, 500); err != nil {
		log.Warn("não foi possível carregar previsões para ranking de risco", "err", err)
	} else {
		preds = list
		sort.Slice(preds, func(i, j int) bool { return preds[i].Score < preds[j].Score })
		risk = make([]reports.RiskSite, 0, len(preds))
		for _, p := range preds {
			risk = append(risk, reports.RiskSite{
				TowerID:     p.TowerID,
				Model:       p.Model,
				Score:       p.Score,
				Status:      p.Status,
				Explanation: p.Explanation,
				CreatedAt:   p.CreatedAt,
			})
		}
		log.Info("ranking de risco carregado", "sites", len(risk))
	}

	base := reports.BuildPeriodReport(period, from, to, incidents, deviations, risk)

	if len(preds) > 0 {
		mc := analytics.MonteCarloPortfolio(preds, 30, 5000, 0)
		base.RiskHorizonDays = mc.HorizonDays
		base.ExpectedFailures = mc.ExpectedFailures
		base.P50Failures = mc.P50Failures
		base.P90Failures = mc.P90Failures
		base.ProbAtLeastOneFail = mc.ProbAtLeastOne
		log.Info("risk Monte Carlo",
			"expected", mc.ExpectedFailures,
			"p90", mc.P90Failures,
			"sites", mc.Sites,
		)
	}

	sender := notification.NewResendSender(cfg.ResendAPIKey, cfg.ResendFrom)
	summarizer := reports.NewExecutiveSummarizer(cfg.OllamaURL, cfg.OllamaModel)

	sent := 0
	for _, p := range reports.AllPersonas {
		recipients := reports.RecipientsFromEnv(p)
		if len(recipients) == 0 {
			log.Info("persona sem destinatários — omitida", "persona", p)
			continue
		}

		meta := reports.MetaFor(p)

		execSummary := ""
		if reports.NeedsSummary(p) {
			execSummary = summarizer.Generate(ctx, base, meta)
			log.Info("resumo executivo gerado", "persona", p, "chars", len(execSummary))
		}

		html, err := reports.BuildPersonaHTML(base, meta, execSummary)
		if err != nil {
			log.Error("erro ao gerar HTML", "persona", p, "err", err)
			continue
		}
		xlsx, err := reports.BuildPersonaExcel(base, meta)
		if err != nil {
			log.Error("erro ao gerar Excel", "persona", p, "err", err)
			continue
		}

		err = sender.Send(recipients, base.Subject(meta), html, []notification.Attachment{{
			Filename: base.Filename(meta),
			Data:     xlsx,
			MimeType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		}})
		if err != nil {
			log.Error("erro ao enviar relatório", "persona", p, "err", err)
			continue
		}

		sent++
		log.Info("relatório enviado",
			"persona", p,
			"recipients", len(recipients),
			"period", period,
			"incidents", base.TotalIncidents,
		)
	}

	if sent == 0 {
		log.Error("nenhum relatório enviado — define REPORT_RECIPIENTS_<PERSONA>")
		os.Exit(1)
	}
	log.Info("job de relatórios concluído", "enviados", sent, "period", period)
}
