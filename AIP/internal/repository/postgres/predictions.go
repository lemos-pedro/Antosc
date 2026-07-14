package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type Prediction struct {
	TowerID          string
	Model            string
	Score            float64
	Status           string
	Explanation      string
	PredictionWindow int
	CreatedAt        time.Time
}

type PredictionRepository interface {
	Save(ctx context.Context, p Prediction) error
	LatestByTower(ctx context.Context, towerID string) (*Prediction, error)
}

type predictionRepository struct {
	db *Database
}

func NewPredictionRepository(db *Database) PredictionRepository {
	return &predictionRepository{db: db}
}

func (r *predictionRepository) Save(ctx context.Context, p Prediction) error {
	_, err := r.db.DB.ExecContext(ctx, `
		INSERT INTO ai_predictions
			(tower_id, model_name, score, status, explanation, prediction_window, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, p.TowerID, p.Model, p.Score, p.Status, p.Explanation, p.PredictionWindow, p.CreatedAt)

	return err
}

// LatestByTower devolve a previsão mais recente para a torre, usada pelo
// endpoint GET /predictions/{tower_id}. nil sem erro significa "sem histórico
// suficiente" — nunca inventamos um valor (regra de "no fabricated data").
func (r *predictionRepository) LatestByTower(ctx context.Context, towerID string) (*Prediction, error) {
	row := r.db.DB.QueryRowContext(ctx, `
		SELECT tower_id, model_name, score, status, explanation, prediction_window, created_at
		FROM ai_predictions
		WHERE tower_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, towerID)

	var p Prediction
	err := row.Scan(&p.TowerID, &p.Model, &p.Score, &p.Status, &p.Explanation, &p.PredictionWindow, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &p, nil
}
