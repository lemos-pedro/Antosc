package postgres

import (
	"context"
	"time"
)

type ConsumptionNorm struct {
	ID                string
	TowerID           string
	EquipmentType     string
	ExpectedValue     float64
	Unit              string
	TolerancePercent  float64
	DefinedBy         string
	Active            bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// ConsumptionNormRepository persiste as normas de consumo definidas pelo Controller.
type ConsumptionNormRepository interface {
	Create(ctx context.Context, n ConsumptionNorm) (ConsumptionNorm, error)
	Update(ctx context.Context, id string, n ConsumptionNorm) (ConsumptionNorm, error)
	ListByTower(ctx context.Context, towerID string) ([]ConsumptionNorm, error)
	ListActive(ctx context.Context) ([]ConsumptionNorm, error)
	Deactivate(ctx context.Context, id string) error
}

type consumptionNormRepository struct {
	db *Database
}

func NewConsumptionNormRepository(db *Database) ConsumptionNormRepository {
	return &consumptionNormRepository{db: db}
}

func (r *consumptionNormRepository) Create(ctx context.Context, n ConsumptionNorm) (ConsumptionNorm, error) {
	row := r.db.DB.QueryRowContext(ctx, `
		INSERT INTO consumption_norms
			(tower_id, equipment_type, expected_value, unit, tolerance_percent, defined_by, active)
		VALUES ($1, $2, $3, $4, $5, $6, TRUE)
		RETURNING id, tower_id, equipment_type, expected_value, unit, tolerance_percent, defined_by, active, created_at, updated_at
	`, n.TowerID, n.EquipmentType, n.ExpectedValue, n.Unit, n.TolerancePercent, n.DefinedBy)

	var out ConsumptionNorm
	err := row.Scan(&out.ID, &out.TowerID, &out.EquipmentType, &out.ExpectedValue, &out.Unit,
		&out.TolerancePercent, &out.DefinedBy, &out.Active, &out.CreatedAt, &out.UpdatedAt)
	return out, err
}

// Update altera uma norma existente. Mantém histórico implícito via updated_at;
// se precisares de histórico completo de alterações, considera uma tabela de auditoria separada mais tarde.
func (r *consumptionNormRepository) Update(ctx context.Context, id string, n ConsumptionNorm) (ConsumptionNorm, error) {
	row := r.db.DB.QueryRowContext(ctx, `
		UPDATE consumption_norms
		SET expected_value = $2, unit = $3, tolerance_percent = $4, defined_by = $5, updated_at = NOW()
		WHERE id = $1
		RETURNING id, tower_id, equipment_type, expected_value, unit, tolerance_percent, defined_by, active, created_at, updated_at
	`, id, n.ExpectedValue, n.Unit, n.TolerancePercent, n.DefinedBy)

	var out ConsumptionNorm
	err := row.Scan(&out.ID, &out.TowerID, &out.EquipmentType, &out.ExpectedValue, &out.Unit,
		&out.TolerancePercent, &out.DefinedBy, &out.Active, &out.CreatedAt, &out.UpdatedAt)
	return out, err
}

func (r *consumptionNormRepository) ListByTower(ctx context.Context, towerID string) ([]ConsumptionNorm, error) {
	rows, err := r.db.DB.QueryContext(ctx, `
		SELECT id, tower_id, equipment_type, expected_value, unit, tolerance_percent, defined_by, active, created_at, updated_at
		FROM consumption_norms
		WHERE tower_id = $1 AND active = TRUE
		ORDER BY equipment_type
	`, towerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNorms(rows)
}

func (r *consumptionNormRepository) ListActive(ctx context.Context) ([]ConsumptionNorm, error) {
	rows, err := r.db.DB.QueryContext(ctx, `
		SELECT id, tower_id, equipment_type, expected_value, unit, tolerance_percent, defined_by, active, created_at, updated_at
		FROM consumption_norms
		WHERE active = TRUE
		ORDER BY tower_id, equipment_type
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNorms(rows)
}

func (r *consumptionNormRepository) Deactivate(ctx context.Context, id string) error {
	_, err := r.db.DB.ExecContext(ctx, `
		UPDATE consumption_norms SET active = FALSE, updated_at = NOW() WHERE id = $1
	`, id)
	return err
}

func scanNorms(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]ConsumptionNorm, error) {
	var out []ConsumptionNorm
	for rows.Next() {
		var n ConsumptionNorm
		if err := rows.Scan(&n.ID, &n.TowerID, &n.EquipmentType, &n.ExpectedValue, &n.Unit,
			&n.TolerancePercent, &n.DefinedBy, &n.Active, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}
