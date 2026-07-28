package postgres

import (
	"context"
	"encoding/json"
	"time"
)

type ToolCallLog struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
	Result    string         `json:"result"`
}

type AssistantAuditEntry struct {
	ID                string
	Role              string
	Question          string
	ToolsCalled       []ToolCallLog
	FinalAnswer       string
	Grounded          bool
	UngroundedValues  string
	CreatedAt         time.Time
}

type AssistantAuditRepository interface {
	Save(ctx context.Context, e AssistantAuditEntry) error
	ListUngrounded(ctx context.Context, limit int) ([]AssistantAuditEntry, error)
}

type assistantAuditRepository struct {
	db *Database
}

func NewAssistantAuditRepository(db *Database) AssistantAuditRepository {
	return &assistantAuditRepository{db: db}
}

func (r *assistantAuditRepository) Save(ctx context.Context, e AssistantAuditEntry) error {
	toolsJSON, err := json.Marshal(e.ToolsCalled)
	if err != nil {
		return err
	}

	_, err = r.db.DB.ExecContext(ctx, `
		INSERT INTO assistant_audit_log (role, question, tools_called, final_answer, grounded, ungrounded_values)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, e.Role, e.Question, toolsJSON, e.FinalAnswer, e.Grounded, e.UngroundedValues)

	return err
}

// ListUngrounded devolve respostas que a verificação sinalizou como
// potencialmente não apoiadas nos dados -- para revisão humana periódica.
func (r *assistantAuditRepository) ListUngrounded(ctx context.Context, limit int) ([]AssistantAuditEntry, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	rows, err := r.db.DB.QueryContext(ctx, `
		SELECT id, role, question, tools_called, final_answer, grounded, ungrounded_values, created_at
		FROM assistant_audit_log
		WHERE grounded = FALSE
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AssistantAuditEntry
	for rows.Next() {
		var e AssistantAuditEntry
		var toolsJSON []byte
		if err := rows.Scan(&e.ID, &e.Role, &e.Question, &toolsJSON, &e.FinalAnswer, &e.Grounded, &e.UngroundedValues, &e.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(toolsJSON, &e.ToolsCalled)
		out = append(out, e)
	}

	return out, rows.Err()
}
