package postgres

import (
	"context"
	"database/sql"
	"time"
)

type IncidentCause struct {
	ID                 string
	TowerID            string
	EventID            sql.NullString
	IncidentStartedAt  time.Time
	IncidentEndedAt    sql.NullTime
	MLPredictedCause   sql.NullString
	MLConfidence       sql.NullFloat64
	ConfirmedCause     sql.NullString
	ConfirmedBy        sql.NullString
	ConfirmedAt        sql.NullTime
	Status             string
	CreatedAt          time.Time
}

// IncidentCauseRepository liga a causa prevista pelo ML à confirmação/correção do O&M.
type IncidentCauseRepository interface {
	CreateFromPrediction(ctx context.Context, c IncidentCause) (IncidentCause, error)
	Confirm(ctx context.Context, id, confirmedCause, confirmedBy string) (IncidentCause, error)
	ListPendingConfirmation(ctx context.Context) ([]IncidentCause, error)
	ListByPeriod(ctx context.Context, from, to time.Time) ([]IncidentCause, error)
	ListByTower(ctx context.Context, towerID string, limit int) ([]IncidentCause, error)
}

type incidentCauseRepository struct {
	db *Database
}

func NewIncidentCauseRepository(db *Database) IncidentCauseRepository {
	return &incidentCauseRepository{db: db}
}

// CreateFromPrediction regista uma nova causa a partir do output do modelo de ML.
// Fica em status "pending_confirmation" até o O&M confirmar ou corrigir.
func (r *incidentCauseRepository) CreateFromPrediction(ctx context.Context, c IncidentCause) (IncidentCause, error) {
	row := r.db.DB.QueryRowContext(ctx, `
		INSERT INTO site_incident_causes
			(tower_id, event_id, incident_started_at, incident_ended_at, ml_predicted_cause, ml_confidence, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending_confirmation')
		RETURNING id, tower_id, event_id, incident_started_at, incident_ended_at,
			ml_predicted_cause, ml_confidence, confirmed_cause, confirmed_by, confirmed_at, status, created_at
	`, c.TowerID, c.EventID, c.IncidentStartedAt, c.IncidentEndedAt, c.MLPredictedCause, c.MLConfidence)

	var out IncidentCause
	err := scanIncidentCause(row, &out)
	return out, err
}

// Confirm regista a decisão do O&M. Se confirmedCause == causa do ML, status fica "confirmed";
// caso contrário fica "corrected" — este sinal é o que alimenta o retreino futuro do modelo de causas.
func (r *incidentCauseRepository) Confirm(ctx context.Context, id, confirmedCause, confirmedBy string) (IncidentCause, error) {
	row := r.db.DB.QueryRowContext(ctx, `
		UPDATE site_incident_causes
		SET confirmed_cause = $2,
			confirmed_by = $3,
			confirmed_at = NOW(),
			status = CASE WHEN ml_predicted_cause = $2 THEN 'confirmed' ELSE 'corrected' END
		WHERE id = $1
		RETURNING id, tower_id, event_id, incident_started_at, incident_ended_at,
			ml_predicted_cause, ml_confidence, confirmed_cause, confirmed_by, confirmed_at, status, created_at
	`, id, confirmedCause, confirmedBy)

	var out IncidentCause
	err := scanIncidentCause(row, &out)
	return out, err
}

func (r *incidentCauseRepository) ListPendingConfirmation(ctx context.Context) ([]IncidentCause, error) {
	rows, err := r.db.DB.QueryContext(ctx, `
		SELECT id, tower_id, event_id, incident_started_at, incident_ended_at,
			ml_predicted_cause, ml_confidence, confirmed_cause, confirmed_by, confirmed_at, status, created_at
		FROM site_incident_causes
		WHERE status = 'pending_confirmation'
		ORDER BY incident_started_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanIncidentCauses(rows)
}

// ListByPeriod é a base do relatório semanal: sites caídos + causa, num intervalo de tempo.
func (r *incidentCauseRepository) ListByPeriod(ctx context.Context, from, to time.Time) ([]IncidentCause, error) {
	rows, err := r.db.DB.QueryContext(ctx, `
		SELECT id, tower_id, event_id, incident_started_at, incident_ended_at,
			ml_predicted_cause, ml_confidence, confirmed_cause, confirmed_by, confirmed_at, status, created_at
		FROM site_incident_causes
		WHERE incident_started_at >= $1 AND incident_started_at < $2
		ORDER BY incident_started_at DESC
	`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanIncidentCauses(rows)
}

func (r *incidentCauseRepository) ListByTower(ctx context.Context, towerID string, limit int) ([]IncidentCause, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.DB.QueryContext(ctx, `
		SELECT id, tower_id, event_id, incident_started_at, incident_ended_at,
			ml_predicted_cause, ml_confidence, confirmed_cause, confirmed_by, confirmed_at, status, created_at
		FROM site_incident_causes
		WHERE tower_id = $1
		ORDER BY incident_started_at DESC
		LIMIT $2
	`, towerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanIncidentCauses(rows)
}

func scanIncidentCause(row *sql.Row, out *IncidentCause) error {
	return row.Scan(&out.ID, &out.TowerID, &out.EventID, &out.IncidentStartedAt, &out.IncidentEndedAt,
		&out.MLPredictedCause, &out.MLConfidence, &out.ConfirmedCause, &out.ConfirmedBy, &out.ConfirmedAt,
		&out.Status, &out.CreatedAt)
}

func scanIncidentCauses(rows *sql.Rows) ([]IncidentCause, error) {
	var out []IncidentCause
	for rows.Next() {
		var c IncidentCause
		if err := rows.Scan(&c.ID, &c.TowerID, &c.EventID, &c.IncidentStartedAt, &c.IncidentEndedAt,
			&c.MLPredictedCause, &c.MLConfidence, &c.ConfirmedCause, &c.ConfirmedBy, &c.ConfirmedAt,
			&c.Status, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
