package postgres

import (
	"context"
	"strings"
	"time"
)

type Feature struct {
	TowerID   string
	Name      string
	Value     float64
	Unit      string
	CreatedAt time.Time
}

// FeatureRepository define o contrato de persistência de features IA.
// Interface exposta para permitir mocks nos testes de internal/analytics/ingestion.
type FeatureRepository interface {
	Save(ctx context.Context, f Feature) error
	SaveBatch(ctx context.Context, features []Feature) error
	ListByTower(ctx context.Context, towerID string, limit int) ([]Feature, error)
	ListByTowerAndRange(ctx context.Context, towerID, featureName string, from, to time.Time) ([]Feature, error)
}

type featureRepository struct {
	db *Database
}

func NewFeatureRepository(db *Database) FeatureRepository {
	return &featureRepository{db: db}
}

func (r *featureRepository) Save(ctx context.Context, f Feature) error {
	_, err := r.db.DB.ExecContext(ctx, `
		INSERT INTO ai_features (tower_id, feature_name, feature_value, unit, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, f.TowerID, f.Name, f.Value, f.Unit, f.CreatedAt)

	return err
}

// SaveBatch grava várias features numa única query (multi-row INSERT).
// Um ciclo de ingestão pode ter centenas de métricas por torre; inserir uma
// a uma seria centenas de round-trips à base de dados. Isto reduz para um.
// maxBatchRows garante que nunca ultrapassamos o limite de 65535
// parâmetros por query do Postgres (5 colunas por linha -- ver placeholders
// abaixo). 5000 linhas * 5 = 25000 parâmetros, com margem confortável.
// Foi um INSERT único de ~55 torres * histórico de 24h que estourou este
// limite no primeiro ciclo de ingestão (296645 parâmetros).
const maxBatchRows = 5000

func (r *featureRepository) SaveBatch(ctx context.Context, features []Feature) error {
	if len(features) == 0 {
		return nil
	}

	for start := 0; start < len(features); start += maxBatchRows {
		end := start + maxBatchRows
		if end > len(features) {
			end = len(features)
		}
		if err := r.saveBatchChunk(ctx, features[start:end]); err != nil {
			return err
		}
	}

	return nil
}

func (r *featureRepository) saveBatchChunk(ctx context.Context, features []Feature) error {
	var sb strings.Builder
	sb.WriteString(`INSERT INTO ai_features (tower_id, feature_name, feature_value, unit, created_at) VALUES `)

	args := make([]any, 0, len(features)*5)
	for i, f := range features {
		if i > 0 {
			sb.WriteString(",")
		}
		base := i * 5
		sb.WriteString(placeholders(base+1, base+5))
		args = append(args, f.TowerID, f.Name, f.Value, f.Unit, f.CreatedAt)
	}

	_, err := r.db.DB.ExecContext(ctx, sb.String(), args...)
	return err
}

func (r *featureRepository) ListByTowerAndRange(ctx context.Context, towerID, featureName string, from, to time.Time) ([]Feature, error) {
	rows, err := r.db.DB.QueryContext(ctx, `
		SELECT tower_id, feature_name, feature_value, unit, created_at
		FROM ai_features
		WHERE tower_id = $1 AND feature_name = $2 AND created_at >= $3 AND created_at < $4
		ORDER BY created_at ASC
	`, towerID, featureName, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Feature
	for rows.Next() {
		var f Feature
		if err := rows.Scan(&f.TowerID, &f.Name, &f.Value, &f.Unit, &f.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}

	return out, rows.Err()
}

func (r *featureRepository) ListByTower(ctx context.Context, towerID string, limit int) ([]Feature, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	rows, err := r.db.DB.QueryContext(ctx, `
		SELECT tower_id, feature_name, feature_value, unit, created_at
		FROM ai_features
		WHERE tower_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, towerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Feature
	for rows.Next() {
		var f Feature
		if err := rows.Scan(&f.TowerID, &f.Name, &f.Value, &f.Unit, &f.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}

	return out, rows.Err()
}
