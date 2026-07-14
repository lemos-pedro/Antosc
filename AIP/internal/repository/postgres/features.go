package postgres

import (
	"context"
	"database/sql"
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

// batchChunkSize limita quantas linhas vão em cada INSERT multi-row.
// O Postgres aceita no máximo 65535 parâmetros por query; com 5
// parâmetros por linha, o limite teórico seria ~13107 linhas. Usamos
// uma margem bem mais folgada (2000) para não andar coladas ao limite
// e para manter cada query com um tamanho razoável.
const batchChunkSize = 2000

// SaveBatch grava várias features em lotes (multi-row INSERT por lote,
// dentro de uma única transação). Um ciclo de ingestão pode trazer
// dezenas de milhares de features (50+ torres x muitas métricas); um
// único INSERT com todas de uma vez ultrapassa o limite de 65535
// parâmetros do Postgres -- por isso paginamos em lotes de
// batchChunkSize, mantendo tudo atómico via transação (ou grava tudo,
// ou nada, mesmo em vários lotes).
func (r *featureRepository) SaveBatch(ctx context.Context, features []Feature) error {
	if len(features) == 0 {
		return nil
	}

	tx, err := r.db.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() // no-op se já tiver havido Commit

	for start := 0; start < len(features); start += batchChunkSize {
		end := start + batchChunkSize
		if end > len(features) {
			end = len(features)
		}

		if err := saveFeatureChunk(ctx, tx, features[start:end]); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func saveFeatureChunk(ctx context.Context, tx *sql.Tx, chunk []Feature) error {
	var sb strings.Builder
	sb.WriteString(`INSERT INTO ai_features (tower_id, feature_name, feature_value, unit, created_at) VALUES `)

	args := make([]any, 0, len(chunk)*5)
	for i, f := range chunk {
		if i > 0 {
			sb.WriteString(",")
		}
		base := i * 5
		sb.WriteString(placeholders(base+1, base+5))
		args = append(args, f.TowerID, f.Name, f.Value, f.Unit, f.CreatedAt)
	}

	_, err := tx.ExecContext(ctx, sb.String(), args...)
	return err
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
