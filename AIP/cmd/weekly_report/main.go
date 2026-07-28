// cmd/weekly_report gera o relatório semanal e envia-o por email via Resend.
// Corre como job agendado (cron no Linux, Task Scheduler no Windows), não
// como processo contínuo -- é mais simples de agendar e de re-executar
// manualmente se falhar.
package main

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/antosc/aip/internal/config"
	"github.com/antosc/aip/internal/logger"
	"github.com/antosc/aip/internal/notification"
	"github.com/antosc/aip/internal/reports"
	"github.com/antosc/aip/internal/repository/postgres"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.LogLevel)

	if cfg.ResendAPIKey == "" {
		log.Error("RESEND_API_KEY não definido -- não é possível enviar o relatório semanal")
		os.Exit(1)
	}

	recipientsEnv := os.Getenv("WEEKLY_REPORT_RECIPIENTS") // lista separada por vírgulas
	if recipientsEnv == "" {
		log.Error("WEEKLY_REPORT_RECIPIENTS não definido")
		os.Exit(1)
	}
	recipients := strings.Split(recipientsEnv, ",")
	for i := range recipients {
		recipients[i] = strings.TrimSpace(recipients[i])
	}

	db, err := postgres.New(cfg.Database.DSN())
	if err != nil {
		log.Error("ligação à base de dados falhou", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	incidentRepo := postgres.NewIncidentCauseRepository(db)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	to := time.Now()
	from := to.AddDate(0, 0, -7)

	incidents, err := incidentRepo.ListByPeriod(ctx, from, to)
	if err != nil {
		log.Error("erro ao obter incidentes da semana", "err", err)
		os.Exit(1)
	}

	report := reports.WeeklyReport{PeriodFrom: from, PeriodTo: to, Incidents: incidents}

	html, err := reports.BuildWeeklyHTML(report)
	if err != nil {
		log.Error("erro ao gerar HTML do relatório", "err", err)
		os.Exit(1)
	}

	xlsx, err := reports.BuildWeeklyExcel(report)
	if err != nil {
		log.Error("erro ao gerar Excel do relatório", "err", err)
		os.Exit(1)
	}

	sender := notification.NewResendSender(cfg.ResendAPIKey, cfg.ResendFrom)

	err = sender.Send(recipients, reports.WeeklySubject(report), html, []notification.Attachment{{
		Filename: "relatorio_semanal.xlsx",
		Data:     xlsx,
		MimeType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	}})
	if err != nil {
		log.Error("erro ao enviar relatório semanal", "err", err)
		os.Exit(1)
	}

	log.Info("relatório semanal enviado", "recipients", len(recipients), "incidents", len(incidents))
}
