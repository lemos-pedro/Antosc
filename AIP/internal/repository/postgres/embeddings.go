package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/lib/pq"
)

type TowerEmbedding struct {
	TowerID      string
	Dims         int
	Vector       []float64
	ModelVersion string
	UpdatedAt    time.Time
}

type EmbeddingRepository interface {
	Upsert(ctx context.Context, e TowerEmbedding) error
	Get(ctx context.Context, towerID string) (*TowerEmbedding, error)
	ListAll(ctx context.Context) ([]TowerEmbedding, error)
}

type embeddingRepository struct {
	db *Database
}

func NewEmbeddingRepository(db *Database) EmbeddingRepository {
	return &embeddingRepository{db: db}
}

func (r *embeddingRepository) Upsert(ctx context.Context, e TowerEmbedding) error {
	_, err := r.db.DB.ExecContext(ctx, `
		INSERT INTO tower_embeddings (tower_id, dims, vector, model_version, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (tower_id) DO UPDATE SET
			dims = EXCLUDED.dims,
			vector = EXCLUDED.vector,
			model_version = EXCLUDED.model_version,
			updated_at = NOW()
	`, e.TowerID, e.Dims, pq.Array(e.Vector), e.ModelVersion)
	return err
}

func (r *embeddingRepository) Get(ctx context.Context, towerID string) (*TowerEmbedding, error) {
	row := r.db.DB.QueryRowContext(ctx, `
		SELECT tower_id, dims, vector, model_version, updated_at
		FROM tower_embeddings WHERE tower_id = $1
	`, towerID)
	var e TowerEmbedding
	var vec pq.Float64Array
	err := row.Scan(&e.TowerID, &e.Dims, &vec, &e.ModelVersion, &e.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	e.Vector = []float64(vec)
	return &e, nil
}

func (r *embeddingRepository) ListAll(ctx context.Context) ([]TowerEmbedding, error) {
	rows, err := r.db.DB.QueryContext(ctx, `
		SELECT tower_id, dims, vector, model_version, updated_at FROM tower_embeddings
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TowerEmbedding
	for rows.Next() {
		var e TowerEmbedding
		var vec pq.Float64Array
		if err := rows.Scan(&e.TowerID, &e.Dims, &vec, &e.ModelVersion, &e.UpdatedAt); err != nil {
			return nil, err
		}
		e.Vector = []float64(vec)
		out = append(out, e)
	}
	return out, rows.Err()
}
