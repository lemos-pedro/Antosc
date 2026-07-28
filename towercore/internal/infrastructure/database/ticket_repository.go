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

type TicketRepository struct {
	db *sql.DB
}

func NewTicketRepository(db *sql.DB) *TicketRepository {
	return &TicketRepository{db: db}
}

func (r *TicketRepository) Create(ctx context.Context, t *domain.Ticket) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const query = `
INSERT INTO tickets (ticket_id, tower_id, event_id, status, created_at, updated_at)
VALUES ($1::uuid, $2::uuid, NULLIF($3, '')::uuid, $4, $5, $6)`

	_, err := r.db.ExecContext(ctx, query,
		t.ID,
		t.TowerID,
		t.EventID,
		string(t.Status),
		t.CreatedAt,
		t.UpdatedAt,
	)
	return err
}

func (r *TicketRepository) GetByID(ctx context.Context, id string) (*domain.Ticket, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const query = `
SELECT
	t.ticket_id::text,
	t.tower_id::text,
	COALESCE(tw.name, ''),
	COALESCE(t.event_id::text, ''),
	t.status,
	t.acknowledged_at,
	t.closed_at,
	t.created_at,
	t.updated_at
FROM tickets t
LEFT JOIN towers tw ON tw.tower_id = t.tower_id
WHERE t.ticket_id::text = $1`

	var t domain.Ticket
	var status string
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&t.ID,
		&t.TowerID,
		&t.TowerName,
		&t.EventID,
		&status,
		&t.AcknowledgedAt,
		&t.ClosedAt,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, interfaces.ErrTicketNotFound
		}
		return nil, err
	}
	t.Status = domain.TicketStatus(status)
	return &t, nil
}

func (r *TicketRepository) List(ctx context.Context, filter interfaces.TicketFilter) ([]domain.Ticket, int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	clauses := make([]string, 0, 2)
	args := make([]any, 0, 4)
	next := 1

	if filter.Status != "" {
		clauses = append(clauses, fmt.Sprintf("t.status = $%d", next))
		args = append(args, filter.Status)
		next++
	}
	if filter.TowerID != "" {
		clauses = append(clauses, fmt.Sprintf("t.tower_id::text = $%d", next))
		args = append(args, filter.TowerID)
		next++
	}

	where := ""
	if len(clauses) > 0 {
		where = " WHERE " + strings.Join(clauses, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM tickets t" + where
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// api.md: "Lista tickets ordenados por data de criacao (desc)"
	listQuery := `
SELECT
	t.ticket_id::text,
	t.tower_id::text,
	COALESCE(tw.name, ''),
	COALESCE(t.event_id::text, ''),
	t.status,
	t.acknowledged_at,
	t.closed_at,
	t.created_at,
	t.updated_at
FROM tickets t
LEFT JOIN towers tw ON tw.tower_id = t.tower_id` + where + fmt.Sprintf(" ORDER BY t.created_at DESC LIMIT $%d OFFSET $%d", next, next+1)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	tickets := make([]domain.Ticket, 0)
	for rows.Next() {
		var t domain.Ticket
		var status string
		if err := rows.Scan(
			&t.ID,
			&t.TowerID,
			&t.TowerName,
			&t.EventID,
			&status,
			&t.AcknowledgedAt,
			&t.ClosedAt,
			&t.CreatedAt,
			&t.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		t.Status = domain.TicketStatus(status)
		tickets = append(tickets, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return tickets, total, nil
}

func (r *TicketRepository) UpdateStatus(
	ctx context.Context,
	id string,
	status domain.TicketStatus,
	acknowledgedAt, closedAt *time.Time,
	updatedAt time.Time,
) (*domain.Ticket, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const query = `
UPDATE tickets SET
	status = $1,
	acknowledged_at = COALESCE($2, acknowledged_at),
	closed_at = COALESCE($3, closed_at),
	updated_at = $4
WHERE ticket_id::text = $5
RETURNING
	ticket_id::text,
	tower_id::text,
	COALESCE(event_id::text, ''),
	status,
	acknowledged_at,
	closed_at,
	created_at,
	updated_at`

	var t domain.Ticket
	var statusStr string
	err := r.db.QueryRowContext(ctx, query, string(status), acknowledgedAt, closedAt, updatedAt, id).Scan(
		&t.ID,
		&t.TowerID,
		&t.EventID,
		&statusStr,
		&t.AcknowledgedAt,
		&t.ClosedAt,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, interfaces.ErrTicketNotFound
		}
		return nil, err
	}
	t.Status = domain.TicketStatus(statusStr)
	return &t, nil
}