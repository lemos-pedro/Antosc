package reports

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ExecutiveSummarizer gera um parágrafo executivo a partir dos KPIs reais
// do PeriodReport, via Ollama. Nunca inventa números — só reformula o que
// recebe no prompt (mesma regra "no fabricated data" do assistente).
//
// Se Ollama estiver indisponível, devolve um fallback determinístico
// construído só com os contadores do relatório.
type ExecutiveSummarizer struct {
	ollamaURL string
	model     string
	http      *http.Client
}

func NewExecutiveSummarizer(ollamaURL, model string) *ExecutiveSummarizer {
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}
	if model == "" {
		model = "qwen2.5:3b"
	}
	return &ExecutiveSummarizer{
		ollamaURL: ollamaURL,
		model:     model,
		http:      &http.Client{Timeout: 60 * time.Second},
	}
}

// NeedsSummary indica se a persona deve receber parágrafo executivo (CEO / Conselho).
func NeedsSummary(p Persona) bool {
	return p == PersonaCEO || p == PersonaConselho
}

// Generate produz 2-4 frases em português, tom executivo.
func (s *ExecutiveSummarizer) Generate(ctx context.Context, r PeriodReport, meta PersonaMeta) string {
	fallback := deterministicSummary(r, meta)

	prompt := buildSummaryPrompt(r, meta)
	text, err := s.chat(ctx, prompt)
	if err != nil || strings.TrimSpace(text) == "" {
		return fallback
	}
	// Segurança extra: se o modelo inventar um número que não está nos
	// factos, preferimos o fallback. Verificação simples de contagens.
	if !numbersGrounded(text, r) {
		return fallback
	}
	return strings.TrimSpace(text)
}

func deterministicSummary(r PeriodReport, meta PersonaMeta) string {
	if r.TotalIncidents == 0 {
		return fmt.Sprintf(
			"No período de %s a %s não se registaram quedas de site. A operação manteve-se estável sem incidentes reportados ao AIP.",
			r.PeriodFrom.Format("02/01"), r.PeriodTo.Format("02/01"),
		)
	}
	tone := "A rede regista"
	if meta.ID == PersonaConselho {
		tone = "Do ponto de vista agregado, a rede regista"
	}
	return fmt.Sprintf(
		"%s %d incidente(s) em %d site(s) entre %s e %s. Destes, %d aguardam confirmação do O&M, %d foram confirmados e %d corrigidos relativamente à previsão do modelo. O ranking IA assinala %d site(s) em risco no snapshot atual. O detalhe está no anexo Excel e pode ser aprofundado via assistente AIP (role=%s).",
		tone,
		r.TotalIncidents, r.SitesAffected,
		r.PeriodFrom.Format("02/01"), r.PeriodTo.Format("02/01"),
		r.PendingConfirmation, r.Confirmed, r.Corrected, r.AtRiskCount,
		meta.ID,
	)
}

func buildSummaryPrompt(r PeriodReport, meta PersonaMeta) string {
	var b strings.Builder
	b.WriteString("Escreve um resumo executivo em português (2 a 4 frases no máximo). ")
	b.WriteString("Usa APENAS os números e factos abaixo — não inventes causas, datas nem quantidades. ")
	b.WriteString("Tom formal e objetivo, adequado a ")
	b.WriteString(meta.DisplayName)
	b.WriteString(".\n\nFACTOS:\n")
	fmt.Fprintf(&b, "- Período: %s a %s\n", r.PeriodFrom.Format("2006-01-02"), r.PeriodTo.Format("2006-01-02"))
	fmt.Fprintf(&b, "- Total de incidentes: %d\n", r.TotalIncidents)
	fmt.Fprintf(&b, "- Sites afetados: %d\n", r.SitesAffected)
	fmt.Fprintf(&b, "- Pendentes de confirmação O&M: %d\n", r.PendingConfirmation)
	fmt.Fprintf(&b, "- Confirmados: %d\n", r.Confirmed)
	fmt.Fprintf(&b, "- Corrigidos (ML diferente de O&M): %d\n", r.Corrected)
	fmt.Fprintf(&b, "- Sites em risco (previsão IA): %d\n", r.AtRiskCount)
	if len(r.Incidents) > 0 {
		b.WriteString("- Exemplos de sites (até 5):\n")
		n := 5
		if len(r.Incidents) < n {
			n = len(r.Incidents)
		}
		for i := 0; i < n; i++ {
			c := r.Incidents[i]
			cause := "sem causa ML"
			if c.MLPredictedCause.Valid {
				cause = c.MLPredictedCause.String
			}
			fmt.Fprintf(&b, "  • %s — %s — estado %s\n", c.TowerID, cause, c.Status)
		}
	}
	b.WriteString("\nResponde só com o parágrafo, sem títulos nem bullets.")
	return b.String()
}

// numbersGrounded verifica se inteiros “grandes” citados no texto aparecem
// nos KPIs. Números 0-10 são ignorados (aparecem em prosa).
func numbersGrounded(text string, r PeriodReport) bool {
	allowed := map[string]bool{
		fmt.Sprintf("%d", r.TotalIncidents):      true,
		fmt.Sprintf("%d", r.SitesAffected):       true,
		fmt.Sprintf("%d", r.PendingConfirmation): true,
		fmt.Sprintf("%d", r.Confirmed):           true,
		fmt.Sprintf("%d", r.Corrected):           true,
	}
	// Extrai sequências de dígitos com 2+ caracteres (evita 1, 2, 3 de frases).
	var num strings.Builder
	flush := func() {
		s := num.String()
		num.Reset()
		if len(s) < 2 {
			return
		}
		if !allowed[s] {
			// data-like (2026, 01, 02) — permitir
			if len(s) == 4 || len(s) == 2 {
				return
			}
			allowed["_fail"] = true
		}
	}
	for _, ch := range text {
		if ch >= '0' && ch <= '9' {
			num.WriteRune(ch)
		} else {
			flush()
		}
	}
	flush()
	return !allowed["_fail"]
}

type ollamaChatReq struct {
	Model    string              `json:"model"`
	Messages []map[string]string `json:"messages"`
	Stream   bool                `json:"stream"`
	Options  map[string]any      `json:"options,omitempty"`
}

type ollamaChatResp struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
}

func (s *ExecutiveSummarizer) chat(ctx context.Context, userPrompt string) (string, error) {
	body, err := json.Marshal(ollamaChatReq{
		Model: s.model,
		Messages: []map[string]string{
			{"role": "system", "content": "És redator de resumos executivos para telecomunicações. Nunca inventas números."},
			{"role": "user", "content": userPrompt},
		},
		Stream:  false,
		Options: map[string]any{"temperature": 0.2},
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.ollamaURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama status %d: %s", resp.StatusCode, string(b))
	}

	var out ollamaChatResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.Message.Content, nil
}
