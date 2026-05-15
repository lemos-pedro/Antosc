package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
)

type AuditRepository struct {
	db *sql.DB
}

func NewAuditRepository(db *sql.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) Create(ctx context.Context, entry *domain.AuditLog) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const query = `
INSERT INTO audit_logs (
	audit_id, actor, action, resource, resource_id, details, created_at
) VALUES (
	$1::uuid, $2, $3, $4, $5, $6, $7
)`

	_, err := r.db.ExecContext(
		ctx,
		query,
		entry.ID,
		entry.Actor,
		entry.Action,
		entry.Resource,
		entry.ResourceID,
		entry.Details,
		entry.CreatedAt,
	)
	return err
}

func (r *AuditRepository) List(ctx context.Context, filter interfaces.AuditFilter) ([]domain.AuditLog, int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	clauses := make([]string, 0, 5)
	args := make([]any, 0, 10)
	next := 1

	if filter.Actor != "" {
		clauses = append(clauses, fmt.Sprintf("actor = $%d", next))
		args = append(args, filter.Actor)
		next++
	}
	if filter.Action != "" {
		clauses = append(clauses, fmt.Sprintf("action = $%d", next))
		args = append(args, filter.Action)
		next++
	}
	if filter.Resource != "" {
		clauses = append(clauses, fmt.Sprintf("resource = $%d", next))
		args = append(args, filter.Resource)
		next++
	}
	if filter.From != nil {
		clauses = append(clauses, fmt.Sprintf("created_at >= $%d", next))
		args = append(args, *filter.From)
		next++
	}
	if filter.To != nil {
		clauses = append(clauses, fmt.Sprintf("created_at <= $%d", next))
		args = append(args, *filter.To)
		next++
	}

	where := ""
	if len(clauses) > 0 {
		where = " WHERE " + strings.Join(clauses, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM audit_logs" + where
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery := `
SELECT
	audit_id::text,
	actor,
	action,
	resource,
	resource_id,
	details,
	created_at
FROM audit_logs` + where + fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", next, next+1)

	args = append(args, filter.Limit, filter.Offset)
	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]domain.AuditLog, 0)
	for rows.Next() {
		var a domain.AuditLog
		if err := rows.Scan(
			&a.ID,
			&a.Actor,
			&a.Action,
			&a.Resource,
			&a.ResourceID,
			&a.Details,
			&a.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (r *AuditRepository) GetByID(ctx context.Context, id string) (*domain.AuditLog, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const query = `
SELECT
	audit_id::text,
	actor,
	action,
	resource,
	resource_id,
	details,
	created_at
FROM audit_logs
WHERE audit_id::text = $1`

	var a domain.AuditLog
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&a.ID,
		&a.Actor,
		&a.Action,
		&a.Resource,
		&a.ResourceID,
		&a.Details,
		&a.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}
	return &a, nil
}
