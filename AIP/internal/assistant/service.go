package assistant

import (
	"context"
	"fmt"
	"log/slog"
)

// ToolExecutor executa uma ferramenta pedida pelo modelo e devolve o
// resultado como texto (normalmente JSON compacto vindo de um endpoint HTTP).
// internal/tools.Registry implementa esta interface.
type ToolExecutor interface {
	Definitions() []ToolDefinition
	Execute(ctx context.Context, name string, args map[string]any) (string, error)
}

// AuditLogger grava cada pergunta/resposta para revisão posterior.
// internal/repository/postgres.AssistantAuditRepository implementa isto
// através de um pequeno adaptador no handler, para não obrigar
// internal/assistant a importar o package postgres diretamente.
type AuditLogger interface {
	Log(ctx context.Context, role, question, finalAnswer string, toolCalls []ToolCallRecord, grounded bool, suspects []string) error
}

type ToolCallRecord struct {
	Name      string
	Arguments map[string]any
	Result    string
}

// maxToolIterations evita ciclos infinitos se o modelo insistir em pedir
// ferramentas indefinidamente -- a partir daqui devolvemos o que tivermos.
const maxToolIterations = 5

type Service struct {
	ollama *OllamaClient
	tools  ToolExecutor
	audit  AuditLogger
	log    *slog.Logger
}

func NewService(ollama *OllamaClient, tools ToolExecutor, audit AuditLogger, log *slog.Logger) *Service {
	return &Service{ollama: ollama, tools: tools, audit: audit, log: log}
}

// AskResult inclui não só a resposta mas o resultado da verificação
// anti-alucinação, para o handler poder decidir se avisa o utilizador.
type AskResult struct {
	Answer   string
	Grounded bool
	Suspects []string
}

// Ask responde a uma pergunta livre, com o prompt de sistema já adaptado ao
// perfil de quem pergunta. Deixa o modelo decidir que ferramentas chamar
// (function calling); o Go só executa o que o modelo pedir contra as APIs
// do AIP -- o modelo nunca tem acesso direto à base de dados. No fim,
// verifica se os números citados na resposta vêm mesmo dos dados obtidos, e
// regista tudo em auditoria.
func (s *Service) Ask(ctx context.Context, role, systemPrompt, question string) (AskResult, error) {
	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: question},
	}
 
	tools := s.tools.Definitions()
	var toolLog []ToolCallRecord
	var toolResults []string

	for i := 0; i < maxToolIterations; i++ {
		msg, err := s.ollama.Chat(ctx, messages, tools)
		if err != nil {
			return AskResult{}, err
		}

		if len(msg.ToolCalls) == 0 {
			grounded, suspects := CheckGrounding(msg.Content, toolResults)

			if s.audit != nil {
				if err := s.audit.Log(ctx, role, question, msg.Content, toolLog, grounded, suspects); err != nil {
					s.log.Warn("falha ao gravar auditoria do assistente", "err", err)
				}
			}
			if !grounded {
				s.log.Warn("resposta do assistente com valores não confirmados nos dados", "suspects", suspects, "role", role)
			}

			return AskResult{Answer: msg.Content, Grounded: grounded, Suspects: suspects}, nil
		}

		messages = append(messages, msg)

		for _, call := range msg.ToolCalls {
			result, err := s.tools.Execute(ctx, call.Function.Name, call.Function.Arguments)
			if err != nil {
				s.log.Warn("falha ao executar ferramenta", "tool", call.Function.Name, "err", err)
				result = fmt.Sprintf(`{"error": %q}`, err.Error())
			}

			toolLog = append(toolLog, ToolCallRecord{Name: call.Function.Name, Arguments: call.Function.Arguments, Result: result})
			toolResults = append(toolResults, result)

			messages = append(messages, ChatMessage{
				Role:     "tool",
				ToolName: call.Function.Name,
				Content:  result,
			})
		}
	}

	return AskResult{}, fmt.Errorf("número máximo de chamadas a ferramentas excedido sem resposta final")
}
