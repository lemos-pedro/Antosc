package postgres

import (
	"context"
	"time"
)

type AuthAuditEntry struct {
	ID        string
	UserID    *string
	Email     string
	Event     string
	IP        string
	UserAgent string
	Detail    string
	CreatedAt time.Time
}

type AuthAuditRepository interface {
	Log(ctx context.Context, e AuthAuditEntry) error
	List(ctx context.Context, limit int) ([]AuthAuditEntry, error)
}

type authAuditRepository struct {
	db *Database
}

func NewAuthAuditRepository(db *Database) AuthAuditRepository {
	return &authAuditRepository{db: db}
}

func (r *authAuditRepository) Log(ctx context.Context, e AuthAuditEntry) error {
	_, err := r.db.DB.ExecContext(ctx, `
		INSERT INTO auth_audit_log (user_id, email, event, ip, user_agent, detail)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, e.UserID, e.Email, e.Event, e.IP, e.UserAgent, e.Detail)
	return err
}

func (r *authAuditRepository) List(ctx context.Context, limit int) ([]AuthAuditEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.DB.QueryContext(ctx, `
		SELECT id, user_id, email, event, ip, user_agent, detail, created_at
		FROM auth_audit_log
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AuthAuditEntry
	for rows.Next() {
		var e AuthAuditEntry
		var uid *string
		if err := rows.Scan(&e.ID, &uid, &e.Email, &e.Event, &e.IP, &e.UserAgent, &e.Detail, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.UserID = uid
		out = append(out, e)
	}
	return out, rows.Err()
}
