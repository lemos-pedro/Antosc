package postgres

import (
	"context"
)

// ListLatestPerTower devolve a previsão mais recente de cada torre
// (DISTINCT ON tower_id). Usado pelo endpoint Power BI /predictions.
// limit protege contra fleets muito grandes; 0 = default 500.
func (r *predictionRepository) ListLatestPerTower(ctx context.Context, limit int) ([]Prediction, error) {
	if limit <= 0 || limit > 2000 {
		limit = 500
	}

	rows, err := r.db.DB.QueryContext(ctx, `
		SELECT DISTINCT ON (tower_id)
			tower_id, model_name, score, status, explanation, prediction_window, created_at
		FROM ai_predictions
		ORDER BY tower_id, created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Prediction
	for rows.Next() {
		var p Prediction
		if err := rows.Scan(
			&p.TowerID, &p.Model, &p.Score, &p.Status,
			&p.Explanation, &p.PredictionWindow, &p.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
