package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
)

type EventRepository struct {
	db *sql.DB
}

func NewEventRepository(db *sql.DB) *EventRepository {
	return &EventRepository{db: db}
}

func (r *EventRepository) Create(ctx context.Context, event *domain.Event) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const query = `
INSERT INTO events (
	event_id, tower_id, type, severity, message, occurred_at, created_at
) VALUES (
	$1::uuid, $2::uuid, $3, $4, $5, $6, $7
)`

	_, err := r.db.ExecContext(
		ctx,
		query,
		event.ID,
		event.TowerID,
		string(event.Type),
		string(event.Severity),
		event.Message,
		event.OccurredAt,
		event.CreatedAt,
	)
	return err
}

func (r *EventRepository) List(ctx context.Context, filter interfaces.EventFilter) ([]domain.Event, int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	clauses := make([]string, 0, 3)
	args := make([]any, 0, 8)
	next := 1

	if filter.TowerID != "" {
		clauses = append(clauses, fmt.Sprintf("tower_id::text = $%d", next))
		args = append(args, filter.TowerID)
		next++
	}
	if filter.Type != "" {
		clauses = append(clauses, fmt.Sprintf("type = $%d", next))
		args = append(args, filter.Type)
		next++
	}
	if filter.Severity != "" {
		clauses = append(clauses, fmt.Sprintf("severity = $%d", next))
		args = append(args, filter.Severity)
		next++
	}

	where := ""
	if len(clauses) > 0 {
		where = " WHERE " + strings.Join(clauses, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM events" + where
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery := `
SELECT
	event_id::text,
	tower_id::text,
	type,
	severity,
	message,
	occurred_at,
	created_at
FROM events` + where + fmt.Sprintf(" ORDER BY occurred_at DESC LIMIT $%d OFFSET $%d", next, next+1)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]domain.Event, 0)
	for rows.Next() {
		var ev domain.Event
		var t, sev string
		if err := rows.Scan(
			&ev.ID,
			&ev.TowerID,
			&t,
			&sev,
			&ev.Message,
			&ev.OccurredAt,
			&ev.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		ev.Type = domain.EventType(t)
		ev.Severity = domain.EventSeverity(sev)
		out = append(out, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return out, total, nil
}
