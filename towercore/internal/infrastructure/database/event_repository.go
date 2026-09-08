package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
)

type EventRepository struct {
	db *sql.DB
}

func NewEventRepository(db *sql.DB) *EventRepository {
	return &EventRepository{db: db}
}

// Create grava um evento novo. Usado tanto pelo fluxo legado (Nagios, sem
// alarm_key) como pelo CreateOrTouch do EventService (SNMP, com alarm_key).
func (r *EventRepository) Create(ctx context.Context, event *domain.Event) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const query = `
INSERT INTO events (
    event_id,
    tower_id,
    type,
    severity,
    message,
    data_source,
    occurred_at,
    created_at,
    status,
    alarm_key,
    last_seen_at
) VALUES (
    $1::uuid,
    $2::uuid,
    $3,
    $4,
    $5,
    COALESCE(NULLIF($6, ''), 'direct_snmp'),
    $7,
    $8,
    $9,
    NULLIF($10, ''),
    $11
)`

	status := string(event.Status)
	if status == "" {
		status = "open"
	}
	lastSeen := event.LastSeenAt
	if lastSeen.IsZero() {
		lastSeen = event.CreatedAt
	}

	_, err := r.db.ExecContext(
		ctx,
		query,
		event.ID,
		event.TowerID,
		string(event.Type),
		string(event.Severity),
		event.Message,
		event.DataSource,
		event.OccurredAt,
		event.CreatedAt,
		status,
		event.AlarmKey,
		lastSeen,
	)
	return err
}

// List agora filtra por status quando pedido, e devolve status/resolved_at
// para que o consumidor (API/frontend) possa distinguir alarmes ativos de
// histórico. ANTES: não filtrava por status nem selecionava a coluna,
// devolvendo eventos resolvidos misturados com abertos — é por isso que a
// UI de "Alarmes" continuava a mostrar eventos já resolvidos pelo
// EventService.Resolve().
func (r *EventRepository) List(ctx context.Context, filter interfaces.EventFilter) ([]domain.Event, int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	clauses := make([]string, 0, 4)
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
	if filter.Status != "" {
		clauses = append(clauses, fmt.Sprintf("status = $%d", next))
		args = append(args, filter.Status)
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
	data_source,
	occurred_at,
	created_at,
	status,
	COALESCE(alarm_key, ''),
	resolved_at
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
		var t, sev, status string
		var resolvedAt sql.NullTime
		if err := rows.Scan(
			&ev.ID,
			&ev.TowerID,
			&t,
			&sev,
			&ev.Message,
			&ev.DataSource,
			&ev.OccurredAt,
			&ev.CreatedAt,
			&status,
			&ev.AlarmKey,
			&resolvedAt,
		); err != nil {
			return nil, 0, err
		}
		ev.Type = domain.EventType(t)
		ev.Severity = domain.EventSeverity(sev)
		ev.Status = domain.EventStatus(status)
		if resolvedAt.Valid {
			ev.ResolvedAt = &resolvedAt.Time
		}
		out = append(out, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return out, total, nil
}

func (r *EventRepository) FindOpenByTowerAndAlarmKey(ctx context.Context, towerID, alarmKey string) (*domain.Event, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const query = `
SELECT event_id::text, tower_id::text, type, severity, message, occurred_at,
       created_at, status, alarm_key, resolved_at, last_seen_at
FROM events
WHERE tower_id::text = $1 AND alarm_key = $2 AND status = 'open'
LIMIT 1`

	var e domain.Event
	var alarmKeyVal sql.NullString
	var resolvedAt sql.NullTime
	err := r.db.QueryRowContext(ctx, query, towerID, alarmKey).Scan(
		&e.ID, &e.TowerID, &e.Type, &e.Severity, &e.Message, &e.OccurredAt,
		&e.CreatedAt, &e.Status, &alarmKeyVal, &resolvedAt, &e.LastSeenAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, interfaces.ErrEventNotFound
		}
		return nil, err
	}
	e.AlarmKey = alarmKeyVal.String
	if resolvedAt.Valid {
		e.ResolvedAt = &resolvedAt.Time
	}
	return &e, nil
}

func (r *EventRepository) TouchLastSeen(ctx context.Context, eventID string, ts time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const query = `UPDATE events SET last_seen_at = $1 WHERE event_id::text = $2`
	_, err := r.db.ExecContext(ctx, query, ts, eventID)
	return err
}

func (r *EventRepository) ResolveOpenByTowerAndAlarmKey(ctx context.Context, towerID, alarmKey string, resolvedAt time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const query = `
UPDATE events
SET status = 'resolved', resolved_at = $1
WHERE tower_id::text = $2 AND alarm_key = $3 AND status = 'open'`
	_, err := r.db.ExecContext(ctx, query, resolvedAt, towerID, alarmKey)
	return err // não é erro se 0 linhas afetadas — idempotente
}

// SumFailureDowntime soma a duração (em segundos) de todos os eventos
// type=failure cuja janela [occurred_at, resolved_at ou windowEnd se
// ainda aberto] se sobrepõe a [windowStart, windowEnd]. Usa
// GREATEST/LEAST para recortar cada evento aos limites da janela pedida
// — um evento que começou antes de windowStart ou que ainda está aberto
// (resolved_at NULL) é contabilizado apenas na parte que cai dentro da
// janela.
func (r *EventRepository) SumFailureDowntime(ctx context.Context, towerID string, windowStart, windowEnd time.Time) (float64, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const query = `
SELECT COALESCE(SUM(
	EXTRACT(EPOCH FROM (
		LEAST(COALESCE(resolved_at, $3), $3) - GREATEST(occurred_at, $2)
	))
), 0)
FROM events
WHERE tower_id::text = $1
  AND type = 'failure'
  AND occurred_at < $3
  AND COALESCE(resolved_at, $3) > $2`

	var seconds float64
	err := r.db.QueryRowContext(ctx, query, towerID, windowStart, windowEnd).Scan(&seconds)
	if err != nil {
		return 0, err
	}
	return seconds, nil
}
