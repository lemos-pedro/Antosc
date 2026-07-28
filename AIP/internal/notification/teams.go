package notification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// TeamsNotifier envia mensagens para um canal do Teams via Incoming Webhook.
// Reservado só para alertas de limiar em tempo real (bateria em desgaste,
// combustível abaixo do mínimo, etc.) -- o relatório semanal completo vai
// por email, não satura o canal com detalhe.
type TeamsNotifier interface {
	SendAlert(alert Alert) error
}

// Alert é um aviso pontual de limiar ultrapassado. Severity: "warning" | "critical".
type Alert struct {
	TowerID   string
	Title     string // ex: "Combustível abaixo do mínimo"
	Detail    string // ex: "Gerador do site X com 42L (mínimo definido: 100L)"
	Severity  string
	Timestamp time.Time
}

type teamsNotifier struct {
	webhookURL string
	http       *http.Client
}

func NewTeamsNotifier(webhookURL string) TeamsNotifier {
	return &teamsNotifier{
		webhookURL: webhookURL,
		http:       &http.Client{Timeout: 10 * time.Second},
	}
}

// teamsCard usa o formato clássico do Office 365 Connector (MessageCard),
// que continua a ser o mais simples de gerar com um webhook de entrada
// direto no canal, sem precisar de registar uma app no Teams.
type teamsCard struct {
	Type       string       `json:"@type"`
	Context    string       `json:"@context"`
	ThemeColor string       `json:"themeColor"`
	Summary    string       `json:"summary"`
	Title      string       `json:"title"`
	Sections   []teamsBlock `json:"sections"`
}

type teamsBlock struct {
	ActivityTitle    string     `json:"activityTitle"`
	ActivitySubtitle string     `json:"activitySubtitle"`
	Facts            []teamsFact `json:"facts"`
}

type teamsFact struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (n *teamsNotifier) SendAlert(alert Alert) error {
	color := "FFA500" // laranja, warning
	if alert.Severity == "critical" {
		color = "D9534F" // vermelho
	}

	card := teamsCard{
		Type:       "MessageCard",
		Context:    "http://schema.org/extensions",
		ThemeColor: color,
		Summary:    alert.Title,
		Title:      "⚠️ " + alert.Title,
		Sections: []teamsBlock{{
			ActivityTitle:    alert.Detail,
			ActivitySubtitle: alert.Timestamp.Format("2006-01-02 15:04"),
			Facts: []teamsFact{
				{Name: "Site", Value: alert.TowerID},
				{Name: "Severidade", Value: alert.Severity},
			},
		}},
	}

	body, err := json.Marshal(card)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, n.webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.http.Do(req)
	if err != nil {
		return fmt.Errorf("teams webhook indisponível: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("teams webhook devolveu status %d", resp.StatusCode)
	}
	return nil
}
