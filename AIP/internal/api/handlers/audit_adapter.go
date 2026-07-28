package handlers

import (
	"context"
	"strings"

	"github.com/antosc/aip/internal/assistant"
	"github.com/antosc/aip/internal/repository/postgres"
)

// auditAdapter implementa assistant.AuditLogger por cima do
// postgres.AssistantAuditRepository, evitando que internal/assistant tenha
// de importar internal/repository/postgres diretamente.
type auditAdapter struct {
	repo postgres.AssistantAuditRepository
}

func NewAuditAdapter(repo postgres.AssistantAuditRepository) assistant.AuditLogger {
	return &auditAdapter{repo: repo}
}

func (a *auditAdapter) Log(ctx context.Context, role, question, finalAnswer string, toolCalls []assistant.ToolCallRecord, grounded bool, suspects []string) error {
	logs := make([]postgres.ToolCallLog, 0, len(toolCalls))
	for _, tc := range toolCalls {
		logs = append(logs, postgres.ToolCallLog{
			Name:      tc.Name,
			Arguments: tc.Arguments,
			Result:    tc.Result,
		})
	}

	return a.repo.Save(ctx, postgres.AssistantAuditEntry{
		Role:             role,
		Question:         question,
		ToolsCalled:      logs,
		FinalAnswer:      finalAnswer,
		Grounded:         grounded,
		UngroundedValues: strings.Join(suspects, ", "),
	})
}
