// Package alerts avalia limiares simples (bateria em desgaste, combustível
// abaixo do mínimo, etc.) a cada ciclo de ingestão e dispara avisos para o
// Teams. É deliberadamente separado do motor preditivo (ML) -- estes são
// alertas imediatos baseados em regra fixa, não previsões de médio prazo.
package alerts

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/antosc/aip/internal/notification"
)

// Rule é um limiar simples: se o valor da feature ultrapassar (para cima ou
// para baixo, consoante Direction) o limite, dispara um alerta.
type Rule struct {
	FeatureName string
	Direction   string // "below" (ex: combustível) ou "above" (ex: temperatura)
	Threshold   float64
	Title       string
	Severity    string // "warning" | "critical"
	Unit        string
}

// DefaultRules cobre os casos que a empresa já identificou como
// prioritários: bateria em desgaste e combustível abaixo do mínimo.
// Ajusta os limiares reais aqui, ou troca por leitura de configuração /
// tabela na BD se precisares de os afinar sem recompilar.
func DefaultRules() []Rule {
	return []Rule{
		{FeatureName: "battery_remaining_percent", Direction: "below", Threshold: 60, Title: "Bateria em desgaste", Severity: "warning", Unit: "%"},
		{FeatureName: "battery_remaining_percent", Direction: "below", Threshold: 40, Title: "Bateria em desgaste crítico", Severity: "critical", Unit: "%"},
		{FeatureName: "fuel_liters", Direction: "below", Threshold: 100, Title: "Combustível abaixo do mínimo", Severity: "warning", Unit: "L"},
		{FeatureName: "fuel_liters", Direction: "below", Threshold: 40, Title: "Combustível crítico", Severity: "critical", Unit: "L"},
	}
}

type Reading struct {
	TowerID     string
	FeatureName string
	Value       float64
}

type Engine struct {
	rules  []Rule
	teams  notification.TeamsNotifier
	log    *slog.Logger
}

func NewEngine(rules []Rule, teams notification.TeamsNotifier, log *slog.Logger) *Engine {
	return &Engine{rules: rules, teams: teams, log: log}
}

// Evaluate corre todas as regras contra as leituras dadas e dispara um
// alerta no Teams para cada uma que ultrapasse o limiar. Falhas de envio são
// só registadas em log -- um alerta que falhe a enviar não deve travar o
// ciclo de ingestão.
func (e *Engine) Evaluate(ctx context.Context, readings []Reading) {
	for _, r := range readings {
		for _, rule := range e.rules {
			if r.FeatureName != rule.FeatureName {
				continue
			}
			if !breaches(r.Value, rule) {
				continue
			}

			alert := notification.Alert{
				TowerID:   r.TowerID,
				Title:     rule.Title,
				Detail:    fmt.Sprintf("%s: %.1f%s (limite: %.1f%s)", rule.Title, r.Value, rule.Unit, rule.Threshold, rule.Unit),
				Severity:  rule.Severity,
				Timestamp: time.Now(),
			}

			if err := e.teams.SendAlert(alert); err != nil {
				e.log.Warn("falha ao enviar alerta para o Teams", "tower_id", r.TowerID, "rule", rule.Title, "err", err)
			}
		}
	}
}

func breaches(value float64, rule Rule) bool {
	if rule.Direction == "below" {
		return value < rule.Threshold
	}
	return value > rule.Threshold
}
