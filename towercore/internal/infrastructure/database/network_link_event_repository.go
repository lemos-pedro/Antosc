package database

import (
	"context"
	"database/sql"
	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
)

type networkLinkEventRepository struct{ db *sql.DB }

func NewNetworkLinkEventRepository(db *sql.DB) interfaces.NetworkLinkEventRepository {
	return &networkLinkEventRepository{db: db}
}

func (r *networkLinkEventRepository) CreateOrTouch(ctx context.Context, e *domain.LinkEvent) error {
	res, err := r.db.ExecContext(ctx, `UPDATE link_events SET type=$2,severity=$3,message=$4,occurred_at=$5 WHERE alarm_key=$1 AND resolved_at IS NULL`, e.AlarmKey, e.Type, e.Severity, e.Message, e.OccurredAt)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return nil
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO link_events (link_id,type,severity,message,occurred_at,alarm_key) VALUES ($1::uuid,$2,$3,$4,$5,$6)`, e.LinkID, e.Type, e.Severity, e.Message, e.OccurredAt, e.AlarmKey)
	return err
}

func (r *networkLinkEventRepository) Resolve(ctx context.Context, alarmKey string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE link_events SET resolved_at=now() WHERE alarm_key=$1 AND resolved_at IS NULL`, alarmKey)
	return err
}

func (r *networkLinkEventRepository) ListByLinkID(ctx context.Context, linkID string, limit, offset int) ([]*domain.LinkEvent, int, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := r.db.QueryContext(ctx, `SELECT event_id::text,link_id::text,type,severity,message,occurred_at,resolved_at,alarm_key FROM link_events WHERE link_id=$1::uuid ORDER BY occurred_at DESC LIMIT $2 OFFSET $3`, linkID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	events := make([]*domain.LinkEvent, 0)
	for rows.Next() {
		e := &domain.LinkEvent{}
		if err := rows.Scan(&e.EventID, &e.LinkID, &e.Type, &e.Severity, &e.Message, &e.OccurredAt, &e.ResolvedAt, &e.AlarmKey); err != nil {
			return nil, 0, err
		}
		events = append(events, e)
	}
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM link_events WHERE link_id=$1::uuid`, linkID).Scan(&total); err != nil {
		return nil, 0, err
	}
	return events, total, rows.Err()
}
