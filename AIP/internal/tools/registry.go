package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/antosc/aip/internal/assistant"
)

// Registry chama as APIs HTTP do próprio AIP em nome do assistente. Nunca
// toca na base de dados diretamente — a API é a única fronteira de acesso
// a dados, o que torna fácil auditar o que o modelo pode ou não ver.
//
// Ferramentas alinhadas com os relatórios por persona e com os datasets
// Power BI: o mesmo dado serve email, BI e Q&A interativo.
type Registry struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func NewRegistry(aipBaseURL, apiKey string) *Registry {
	return &Registry{
		baseURL: aipBaseURL,
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 20 * time.Second},
	}
}

// Definitions devolve o catálogo de ferramentas no formato esperado pelo Ollama.
func (r *Registry) Definitions() []assistant.ToolDefinition {
	return []assistant.ToolDefinition{
		// --- existentes (por site / período) ---
		{
			Type: "function",
			Function: assistant.ToolFunctionSchema{
				Name:        "sites_down_weekly",
				Description: "Lista os sites que caíram num período, com causa ML e/ou confirmada pelo O&M. Usa para relatórios e perguntas sobre quedas.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"from": map[string]any{"type": "string", "description": "Data de início, YYYY-MM-DD"},
						"to":   map[string]any{"type": "string", "description": "Data de fim, YYYY-MM-DD (opcional, default from+7d)"},
					},
					"required": []string{"from"},
				},
			},
		},
		{
			Type: "function",
			Function: assistant.ToolFunctionSchema{
				Name:        "pending_confirmations",
				Description: "Lista incidentes que o O&M ainda não confirmou/corrigiu. Prioridade de ação para perfil O&M e Diretor Técnico.",
				Parameters: map[string]any{
					"type":       "object",
					"properties": map[string]any{},
				},
			},
		},
		{
			Type: "function",
			Function: assistant.ToolFunctionSchema{
				Name:        "site_health",
				Description: "Estado de saúde / previsão mais recente de um site específico.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"tower_id": map[string]any{"type": "string", "description": "ID do site/torre"},
					},
					"required": []string{"tower_id"},
				},
			},
		},
		{
			Type: "function",
			Function: assistant.ToolFunctionSchema{
				Name:        "site_predictions",
				Description: "Previsão de falha mais recente para um site (score, status, explicação, janela).",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"tower_id": map[string]any{"type": "string", "description": "ID do site/torre"},
					},
					"required": []string{"tower_id"},
				},
			},
		},
		{
			Type: "function",
			Function: assistant.ToolFunctionSchema{
				Name:        "consumption_deviations",
				Description: "Desvio de consumo de UM site vs normas do Controller, num período.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"tower_id": map[string]any{"type": "string", "description": "ID do site/torre"},
						"from":     map[string]any{"type": "string", "description": "YYYY-MM-DD"},
						"to":       map[string]any{"type": "string", "description": "YYYY-MM-DD opcional"},
					},
					"required": []string{"tower_id", "from"},
				},
			},
		},
		{
			Type: "function",
			Function: assistant.ToolFunctionSchema{
				Name:        "compare_consumption_periods",
				Description: "Compara consumo médio de um site entre dois períodos.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"tower_id":      map[string]any{"type": "string"},
						"period_a_from": map[string]any{"type": "string"},
						"period_a_to":   map[string]any{"type": "string"},
						"period_b_from": map[string]any{"type": "string"},
						"period_b_to":   map[string]any{"type": "string"},
					},
					"required": []string{"tower_id", "period_a_from", "period_b_from"},
				},
			},
		},
		// --- portfolio / cross-site (novas — alinham com Power BI e emails) ---
		{
			Type: "function",
			Function: assistant.ToolFunctionSchema{
				Name:        "portfolio_kpis",
				Description: "KPIs agregados da rede num período: total de incidentes, sites afetados, pendentes O&M, confirmados, corrigidos, desvios fora de norma. Ideal para CEO, Conselho e Diretor Técnico.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"from": map[string]any{"type": "string", "description": "YYYY-MM-DD"},
						"to":   map[string]any{"type": "string", "description": "YYYY-MM-DD opcional"},
					},
					"required": []string{"from"},
				},
			},
		},
		{
			Type: "function",
			Function: assistant.ToolFunctionSchema{
				Name:        "consumption_portfolio",
				Description: "Desvios de consumo de TODOS os sites com norma ativa, num período. Para Financeiro e Controller (conformidade global).",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"from": map[string]any{"type": "string", "description": "YYYY-MM-DD"},
						"to":   map[string]any{"type": "string", "description": "YYYY-MM-DD opcional"},
					},
					"required": []string{"from"},
				},
			},
		},
		{
			Type: "function",
			Function: assistant.ToolFunctionSchema{
				Name:        "risk_ranking",
				Description: "Previsão IA mais recente de cada torre (score, status, explicação). Usa para ranking de risco e prioridades de intervenção.",
				Parameters: map[string]any{
					"type":       "object",
					"properties": map[string]any{},
				},
			},
		},
		{
			Type: "function",
			Function: assistant.ToolFunctionSchema{
				Name:        "towers_directory",
				Description: "Lista de torres na cache local (vendor, região, disponibilidade 7d/30d). Útil para contextualizar sites ou filtrar por fabricante.",
				Parameters: map[string]any{
					"type":       "object",
					"properties": map[string]any{},
				},
			},
		},
		{
			Type: "function",
			Function: assistant.ToolFunctionSchema{
				Name:        "portfolio_risk",
				Description: "Risk score agregado da rede (Monte Carlo): falhas esperadas, P50/P90, probabilidade de pelo menos uma falha no horizonte.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"horizon_days": map[string]any{"type": "integer", "description": "Horizonte em dias (default 30)"},
						"sims":         map[string]any{"type": "integer", "description": "Nº de simulações (default 5000)"},
					},
				},
			},
		},
		{
			Type: "function",
			Function: assistant.ToolFunctionSchema{
				Name:        "similar_towers",
				Description: "Torres semelhantes via embeddings cross-tower (PCA). Requer embeddings_job.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"tower_id": map[string]any{"type": "string"},
						"k":        map[string]any{"type": "integer", "description": "Top-K (default 5)"},
					},
					"required": []string{"tower_id"},
				},
			},
		},
	}
}

// Execute despacha a ferramenta pedida pelo modelo para o endpoint HTTP correspondente.
func (r *Registry) Execute(ctx context.Context, name string, args map[string]any) (string, error) {
	switch name {
	case "sites_down_weekly":
		return r.get(ctx, "/api/v1/incidents", url.Values{
			"from": {str(args["from"])},
			"to":   {str(args["to"])},
		})
	case "pending_confirmations":
		return r.get(ctx, "/api/v1/incidents/pending", nil)
	case "site_health", "site_predictions":
		return r.get(ctx, "/api/v1/predictions/"+str(args["tower_id"]), nil)
	case "consumption_deviations":
		return r.get(ctx, "/api/v1/consumption/deviations/"+str(args["tower_id"]), url.Values{
			"from": {str(args["from"])},
			"to":   {str(args["to"])},
		})
	case "compare_consumption_periods":
		return r.get(ctx, "/api/v1/consumption/deviations/"+str(args["tower_id"])+"/compare", url.Values{
			"period_a_from": {str(args["period_a_from"])},
			"period_a_to":   {str(args["period_a_to"])},
			"period_b_from": {str(args["period_b_from"])},
			"period_b_to":   {str(args["period_b_to"])},
		})
	case "portfolio_kpis":
		return r.get(ctx, "/api/v1/powerbi/kpis", url.Values{
			"from": {str(args["from"])},
			"to":   {str(args["to"])},
		})
	case "consumption_portfolio":
		return r.get(ctx, "/api/v1/powerbi/consumption", url.Values{
			"from": {str(args["from"])},
			"to":   {str(args["to"])},
		})
	case "risk_ranking":
		return r.get(ctx, "/api/v1/powerbi/predictions", nil)
	case "towers_directory":
		return r.get(ctx, "/api/v1/powerbi/towers", nil)
	case "portfolio_risk":
		q := url.Values{}
		if str(args["horizon_days"]) != "" {
			q.Set("horizon_days", str(args["horizon_days"]))
		}
		if str(args["sims"]) != "" {
			q.Set("sims", str(args["sims"]))
		}
		return r.get(ctx, "/api/v1/risk/portfolio", q)
	case "similar_towers":
		q := url.Values{}
		if str(args["k"]) != "" {
			q.Set("k", str(args["k"]))
		}
		return r.get(ctx, "/api/v1/towers/"+str(args["tower_id"])+"/similar", q)
	default:
		return "", fmt.Errorf("ferramenta desconhecida: %s", name)
	}
}

func (r *Registry) get(ctx context.Context, path string, query url.Values) (string, error) {
	full := r.baseURL + path
	if query != nil {
		clean := url.Values{}
		for k, v := range query {
			if len(v) > 0 && v[0] != "" {
				clean[k] = v
			}
		}
		if len(clean) > 0 {
			full += "?" + clean.Encode()
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, full, nil)
	if err != nil {
		return "", err
	}
	if r.apiKey != "" {
		req.Header.Set("X-API-Key", r.apiKey)
	}

	resp, err := r.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("chamada a %s falhou: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("endpoint %s devolveu status %d: %s", path, resp.StatusCode, string(body))
	}

	var v any
	if err := json.Unmarshal(body, &v); err == nil {
		if compact, err := json.Marshal(v); err == nil {
			return string(compact), nil
		}
	}
	return string(body), nil
}

func str(v any) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}
