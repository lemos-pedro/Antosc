package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
)

type MetricRepository struct {
	db *sql.DB
}

func NewMetricRepository(db *sql.DB) *MetricRepository {
	return &MetricRepository{db: db}
}

func (r *MetricRepository) Create(ctx context.Context, metric *domain.Metric) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	const query = `
INSERT INTO metrics (
	metric_id, tower_id, collected_at, values, created_at
) VALUES (
	$1::uuid, $2::uuid, $3, $4::jsonb, $5
)`

	valuesJSON, err := json.Marshal(metric.Values)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(
		ctx,
		query,
		metric.ID,
		metric.TowerID,
		metric.CollectedAt,
		string(valuesJSON),
		metric.CreatedAt,
	)
	return err
}

func (r *MetricRepository) List(ctx context.Context, filter interfaces.MetricFilter) ([]domain.Metric, int, error) {
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
	if filter.From != nil {
		clauses = append(clauses, fmt.Sprintf("collected_at >= $%d", next))
		args = append(args, *filter.From)
		next++
	}
	if filter.To != nil {
		clauses = append(clauses, fmt.Sprintf("collected_at <= $%d", next))
		args = append(args, *filter.To)
		next++
	}

	where := ""
	if len(clauses) > 0 {
		where = " WHERE " + strings.Join(clauses, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM metrics" + where
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery := `
SELECT
	metric_id::text,
	tower_id::text,
	collected_at,
	values,
	created_at
FROM metrics` + where + fmt.Sprintf(" ORDER BY collected_at DESC LIMIT $%d OFFSET $%d", next, next+1)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]domain.Metric, 0)
	for rows.Next() {
		var m domain.Metric
		var valuesRaw []byte
		if err := rows.Scan(
			&m.ID,
			&m.TowerID,
			&m.CollectedAt,
			&valuesRaw,
			&m.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		if len(valuesRaw) == 0 {
			m.Values = map[string]float64{}
		} else {
			if err := json.Unmarshal(valuesRaw, &m.Values); err != nil {
				return nil, 0, err
			}
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return out, total, nil
}
