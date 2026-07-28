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
type Registry struct {
	baseURL string
	http    *http.Client
}

func NewRegistry(aipBaseURL string) *Registry {
	return &Registry{
		baseURL: aipBaseURL,
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

// Definitions devolve o catálogo de ferramentas no formato esperado pelo Ollama.
// Cada ferramenta espelha um endpoint já existente do AIP — se adicionares um
// endpoint novo, adiciona aqui a ferramenta correspondente.
func (r *Registry) Definitions() []assistant.ToolDefinition {
	return []assistant.ToolDefinition{
		{
			Type: "function",
			Function: assistant.ToolFunctionSchema{
				Name:        "sites_down_weekly",
				Description: "Lista os sites que caíram numa semana, com a causa provável (do ML e/ou confirmada pelo O&M). Usa para relatórios semanais e perguntas sobre quedas de sites.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"from": map[string]any{"type": "string", "description": "Data de início, formato YYYY-MM-DD"},
						"to":   map[string]any{"type": "string", "description": "Data de fim, formato YYYY-MM-DD (opcional, default: from + 7 dias)"},
					},
					"required": []string{"from"},
				},
			},
		},
		{
			Type: "function",
			Function: assistant.ToolFunctionSchema{
				Name:        "site_health",
				Description: "Devolve o estado de saúde atual e a previsão mais recente de um site específico.",
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
				Description: "Compara o consumo real de um site com as normas definidas pelo Controller, num período. Usa para perguntas de conformidade e para o Financeiro.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"tower_id": map[string]any{"type": "string", "description": "ID do site/torre"},
						"from":     map[string]any{"type": "string", "description": "Data de início, formato YYYY-MM-DD"},
						"to":       map[string]any{"type": "string", "description": "Data de fim, formato YYYY-MM-DD (opcional)"},
					},
					"required": []string{"tower_id", "from"},
				},
			},
		},
		{
			Type: "function",
			Function: assistant.ToolFunctionSchema{
				Name:        "site_predictions",
				Description: "Devolve a previsão de falha mais recente para um site (janela estimada e nível de confiança).",
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
				Name:        "compare_consumption_periods",
				Description: "Compara o consumo médio real de um site entre dois períodos diferentes (ex: mês passado vs este mês, ou duas épocas quaisquer).",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"tower_id":       map[string]any{"type": "string", "description": "ID do site/torre"},
						"period_a_from":  map[string]any{"type": "string", "description": "Início do primeiro período, YYYY-MM-DD"},
						"period_a_to":    map[string]any{"type": "string", "description": "Fim do primeiro período, YYYY-MM-DD"},
						"period_b_from":  map[string]any{"type": "string", "description": "Início do segundo período, YYYY-MM-DD"},
						"period_b_to":    map[string]any{"type": "string", "description": "Fim do segundo período, YYYY-MM-DD"},
					},
					"required": []string{"tower_id", "period_a_from", "period_b_from"},
				},
			},
		},
	}
}

// Execute despacha uma chamada de ferramenta pedida pelo modelo para o
// endpoint HTTP correspondente, e devolve o corpo da resposta como string
// (para ser reinserido na conversa como mensagem role="tool").
func (r *Registry) Execute(ctx context.Context, name string, args map[string]any) (string, error) {
	switch name {
	case "sites_down_weekly":
		return r.get(ctx, "/api/v1/incidents", url.Values{
			"from": {str(args["from"])},
			"to":   {str(args["to"])},
		})
	case "site_health":
		return r.get(ctx, "/api/v1/predictions/"+str(args["tower_id"]), nil)
	case "consumption_deviations":
		return r.get(ctx, "/api/v1/consumption/deviations/"+str(args["tower_id"]), url.Values{
			"from": {str(args["from"])},
			"to":   {str(args["to"])},
		})
	case "site_predictions":
		return r.get(ctx, "/api/v1/predictions/"+str(args["tower_id"]), nil)
	case "compare_consumption_periods":
		return r.get(ctx, "/api/v1/consumption/deviations/"+str(args["tower_id"])+"/compare", url.Values{
			"period_a_from": {str(args["period_a_from"])},
			"period_a_to":   {str(args["period_a_to"])},
			"period_b_from": {str(args["period_b_from"])},
			"period_b_to":   {str(args["period_b_to"])},
		})
	default:
		return "", fmt.Errorf("ferramenta desconhecida: %s", name)
	}
}

func (r *Registry) get(ctx context.Context, path string, query url.Values) (string, error) {
	full := r.baseURL + path
	if query != nil {
		// Remove parâmetros vazios (ex: "to" não fornecido pelo modelo) para
		// deixar o handler aplicar o seu próprio default.
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

	// Normaliza para JSON compacto antes de devolver ao modelo -- não é
	// estritamente necessário, mas evita gastar tokens com indentação.
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
