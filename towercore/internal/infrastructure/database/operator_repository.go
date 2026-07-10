package database

import (
	"context"
	"database/sql"

	"towercore/internal/core/domain"
)

type OperatorRepository struct {
	db *sql.DB
}

func NewOperatorRepository(db *sql.DB) *OperatorRepository {
	return &OperatorRepository{db: db}
}

func (r *OperatorRepository) List(ctx context.Context) ([]domain.Operator, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT operator_id, name, code, created_at, updated_at
		FROM operators
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var operators []domain.Operator

	for rows.Next() {
		var op domain.Operator

		if err := rows.Scan(
			&op.OperatorID,
			&op.Name,
			&op.Code,
			&op.CreatedAt,
			&op.UpdatedAt,
		); err != nil {
			return nil, err
		}

		operators = append(operators, op)
	}

	return operators, nil
}

func (r *OperatorRepository) Create(ctx context.Context, operator *domain.Operator) error {
	return r.db.QueryRowContext(
		ctx,
		`
		INSERT INTO operators (name, code)
		VALUES ($1, $2)
		RETURNING operator_id, created_at, updated_at
		`,
		operator.Name,
		operator.Code,
	).Scan(
		&operator.OperatorID,
		&operator.CreatedAt,
		&operator.UpdatedAt,
	)
}