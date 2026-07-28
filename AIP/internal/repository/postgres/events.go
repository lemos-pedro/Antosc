package postgres

import (
	"context"
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

// maxBatchRows está definido em features.go (mesmo package) -- reaproveitado aqui.

func (r *eventRepository) SaveBatch(ctx context.Context, events []AIEvent) error {
	if len(events) == 0 {
		return nil
	}

	for start := 0; start < len(events); start += maxBatchRows {
		end := start + maxBatchRows
		if end > len(events) {
			end = len(events)
		}
		if err := r.saveBatchChunk(ctx, events[start:end]); err != nil {
			return err
		}
	}

	return nil
}

func (r *eventRepository) saveBatchChunk(ctx context.Context, events []AIEvent) error {
	var sb strings.Builder
	sb.WriteString(`INSERT INTO ai_events (tower_id, event_type, severity, message, created_at) VALUES `)

	args := make([]any, 0, len(events)*5)
	for i, e := range events {
		if i > 0 {
			sb.WriteString(",")
		}
		base := i * 5
		sb.WriteString(placeholders(base+1, base+5))
		args = append(args, e.TowerID, e.Type, e.Severity, e.Message, e.CreatedAt)
	}

	_, err := r.db.DB.ExecContext(ctx, sb.String(), args...)
	return err
}
