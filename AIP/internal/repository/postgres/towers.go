package postgres

import (
	"context"
	"database/sql"
)

type Tower struct {
	TowerID          string
	Name             string
	Vendor           string
	OperatorID       string
	RegionID         string
	Availability7d   sql.NullFloat64
	Availability30d  sql.NullFloat64
}

type TowerRepository interface {
	// Upsert atualiza (ou cria) o registo local da torre. Chamado a cada
	// ciclo de ingestão com o que o towercore devolveu -- é uma cache, não
	// a fonte de verdade.
	Upsert(ctx context.Context, t Tower) error
	// UpsertBatch faz o mesmo para várias torres numa só chamada.
	UpsertBatch(ctx context.Context, towers []Tower) error
	// GetVendor devolve o vendor conhecido para uma torre -- usado pelo
	// serviço de previsão para saber que adaptador pedir ao Python.
	GetVendor(ctx context.Context, towerID string) (string, error)
	ListAll(ctx context.Context) ([]Tower, error)
}

type towerRepository struct {
	db *Database
}

func NewTowerRepository(db *Database) TowerRepository {
	return &towerRepository{db: db}
}

func (r *towerRepository) Upsert(ctx context.Context, t Tower) error {
	_, err := r.db.DB.ExecContext(ctx, `
		INSERT INTO towers (tower_id, name, vendor, operator_id, region_id, availability_7d, availability_30d, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (tower_id) DO UPDATE SET
			name = EXCLUDED.name,
			vendor = EXCLUDED.vendor,
			operator_id = EXCLUDED.operator_id,
			region_id = EXCLUDED.region_id,
			availability_7d = EXCLUDED.availability_7d,
			availability_30d = EXCLUDED.availability_30d,
			updated_at = NOW()
	`, t.TowerID, t.Name, t.Vendor, t.OperatorID, t.RegionID, t.Availability7d, t.Availability30d)
	return err
}

func (r *towerRepository) UpsertBatch(ctx context.Context, towers []Tower) error {
	for _, t := range towers {
		if err := r.Upsert(ctx, t); err != nil {
			return err
		}
	}
	return nil
}

func (r *towerRepository) GetVendor(ctx context.Context, towerID string) (string, error) {
	var vendor sql.NullString
	err := r.db.DB.QueryRowContext(ctx, `SELECT vendor FROM towers WHERE tower_id = $1`, towerID).Scan(&vendor)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return vendor.String, nil
}
