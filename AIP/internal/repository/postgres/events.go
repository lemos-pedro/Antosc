package postgres

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

type AIEvent struct {
	TowerID   string
	Type      string
	Severity  string
	Message   string
	CreatedAt time.Time
}

type EventRepository interface {
	Save(ctx context.Context, e AIEvent) error
	SaveBatch(ctx context.Context, events []AIEvent) error

	// CreateOrTouch grava um evento de condição persistente (ex. "torre
	// degraded") sem duplicar: se já existir um evento "open" com a
	// mesma (tower_id, alarm_key), só atualiza last_seen_at/severity/
	// message; caso contrário cria um novo. Isto é o padrão dedup que
	// falta no towercore (Prioridade #1 do projeto) -- aqui aplicado ao
	// que o AIP consegue detetar sozinho a partir do status das torres.
	CreateOrTouch(ctx context.Context, e OpenEvent) error

	// Resolve fecha o evento "open" (se existir) para (tower_id,
	// alarm_key) -- chamado quando a condição deixa de se verificar
	// (ex. torre voltou a "online").
	Resolve(ctx context.Context, towerID, alarmKey string) error
}

// OpenEvent é o input de CreateOrTouch -- inclui AlarmKey, que AIEvent
// (usado no fluxo de ingestão simples, saveEvents) não precisa de ter.
type OpenEvent struct {
	TowerID   string
	AlarmKey  string
	Type      string
	Severity  string
	Message   string
}

type eventRepository struct {
	db *Database
}

func NewEventRepository(db *Database) EventRepository {
	return &eventRepository{db: db}
}

func (r *eventRepository) Save(ctx context.Context, e AIEvent) error {
	_, err := r.db.DB.ExecContext(ctx, `
		INSERT INTO ai_events (tower_id, event_type, severity, message, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, e.TowerID, e.Type, e.Severity, e.Message, e.CreatedAt)

	return err
}

// SaveBatch grava vários eventos em lotes (multi-row INSERT por lote,
// dentro de uma única transação) -- mesma razão e mesmo limite do
// Postgres (65535 parâmetros por query) que em features.go SaveBatch.
func (r *eventRepository) SaveBatch(ctx context.Context, events []AIEvent) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := r.db.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for start := 0; start < len(events); start += batchChunkSize {
		end := start + batchChunkSize
		if end > len(events) {
			end = len(events)
		}

		if err := saveEventChunk(ctx, tx, events[start:end]); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func saveEventChunk(ctx context.Context, tx *sql.Tx, chunk []AIEvent) error {
	var sb strings.Builder
	sb.WriteString(`INSERT INTO ai_events (tower_id, event_type, severity, message, created_at) VALUES `)

	args := make([]any, 0, len(chunk)*5)
	for i, e := range chunk {
		if i > 0 {
			sb.WriteString(",")
		}
		base := i * 5
		sb.WriteString(placeholders(base+1, base+5))
		args = append(args, e.TowerID, e.Type, e.Severity, e.Message, e.CreatedAt)
	}

	_, err := tx.ExecContext(ctx, sb.String(), args...)
	return err
}

// CreateOrTouch usa o índice único parcial idx_ai_events_open_dedup
// (tower_id, alarm_key) WHERE status='open' -- ver migration
// 002_events_dedup.sql. Enquanto a condição persistir, este INSERT
// entra sempre em conflito com a linha "open" já existente e só
// atualiza last_seen_at/severity/message, nunca duplica.
func (r *eventRepository) CreateOrTouch(ctx context.Context, e OpenEvent) error {
	_, err := r.db.DB.ExecContext(ctx, `
		INSERT INTO ai_events
			(tower_id, event_type, alarm_key, severity, message, status, created_at, last_seen_at)
		VALUES
			($1, $2, $3, $4, $5, 'open', NOW(), NOW())
		ON CONFLICT (tower_id, alarm_key) WHERE status = 'open'
		DO UPDATE SET
			last_seen_at = NOW(),
			severity     = EXCLUDED.severity,
			message      = EXCLUDED.message
	`, e.TowerID, e.Type, e.AlarmKey, e.Severity, e.Message)

	return err
}

// Resolve fecha o evento "open" (se existir) para (tower_id, alarm_key).
// Não faz nada (sem erro) se não houver nenhum aberto -- resolver algo
// que já não está aberto é um no-op, não uma falha.
func (r *eventRepository) Resolve(ctx context.Context, towerID, alarmKey string) error {
	_, err := r.db.DB.ExecContext(ctx, `
		UPDATE ai_events
		SET status = 'resolved', resolved_at = NOW()
		WHERE tower_id = $1 AND alarm_key = $2 AND status = 'open'
	`, towerID, alarmKey)

	return err
}
