// Package assistant liga o modelo local (Ollama) às APIs do AIP via
// function calling. O modelo nunca acede à base de dados diretamente —
// só pode "ver" o que os endpoints HTTP do AIP devolverem.
package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// OllamaClient fala com a API nativa do Ollama (POST /api/chat), que suporta
// tool calling para modelos preparados para isso (ex: qwen2.5, llama3.1).
type OllamaClient struct {
	baseURL string
	model   string
	http    *http.Client
}

func NewOllamaClient(baseURL, model string) *OllamaClient {
	return &OllamaClient{
		baseURL: baseURL,
		model:   model,
		// Respostas de modelos locais podem demorar, sobretudo com várias
		// chamadas a ferramentas em sequência; timeout generoso.
		http: &http.Client{Timeout: 300 * time.Second},
	}
}

type ChatMessage struct {
	Role      string     `json:"role"` // "system" | "user" | "assistant" | "tool"
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	// ToolName identifica, numa mensagem role="tool", a que ferramenta
	// corresponde o resultado devolvido em Content.
	ToolName string `json:"tool_name,omitempty"`
}

type ToolCall struct {
	Function ToolCallFunction `json:"function"`
}

type ToolCallFunction struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// ToolDefinition descreve uma ferramenta no formato que o Ollama espera
// (compatível com o esquema de "function calling" da OpenAI).
type ToolDefinition struct {
	Type     string             `json:"type"` // sempre "function"
	Function ToolFunctionSchema `json:"function"`
}

type ToolFunctionSchema struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type chatRequest struct {
	Model    string           `json:"model"`
	Messages []ChatMessage    `json:"messages"`
	Tools    []ToolDefinition `json:"tools,omitempty"`
	Stream   bool             `json:"stream"`
	Options  map[string]any   `json:"options,omitempty"`
}

type chatResponse struct {
	Message ChatMessage `json:"message"`
	Done    bool        `json:"done"`
}

// Chat envia o histórico + ferramentas disponíveis e devolve a mensagem do
// modelo (que pode ser uma resposta final em texto, ou um pedido para
// executar uma ou mais ferramentas via ToolCalls).
func (c *OllamaClient) Chat(ctx context.Context, messages []ChatMessage, tools []ToolDefinition) (ChatMessage, error) {
	reqBody, err := json.Marshal(chatRequest{
		Model:    c.model,
		Messages: messages,
		Tools:    tools,
		Stream:   false,
		// Temperature baixa reduz "criatividade" -- que é literalmente o
		// mecanismo que produz alucinação. Não elimina o risco, mas ajuda
		// bastante em respostas que devem ser factuais.
		Options: map[string]any{"temperature": 0.1},
	})
	if err != nil {
		return ChatMessage{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/chat", bytes.NewReader(reqBody))
	if err != nil {
		return ChatMessage{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return ChatMessage{}, fmt.Errorf("ollama indisponível: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ChatMessage{}, fmt.Errorf("ollama devolveu status %d", resp.StatusCode)
	}

	var out chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return ChatMessage{}, err
	}

	return out.Message, nil
}
