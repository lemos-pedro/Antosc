package database

import (
	"context"
	"database/sql"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
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

func (r *OperatorRepository) GetByID(ctx context.Context, operatorID string) (*domain.Operator, error) {
	var op domain.Operator
	err := r.db.QueryRowContext(ctx, `
		SELECT operator_id, name, code, created_at, updated_at
		FROM operators
		WHERE operator_id::text = $1
	`, operatorID).Scan(
		&op.OperatorID,
		&op.Name,
		&op.Code,
		&op.CreatedAt,
		&op.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, interfaces.ErrOperatorNotFound
		}
		return nil, err
	}
	return &op, nil
}

func (r *OperatorRepository) Update(ctx context.Context, operator *domain.Operator) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE operators
		SET name = $1, code = $2, updated_at = now()
		WHERE operator_id::text = $3
	`, operator.Name, operator.Code, operator.OperatorID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return interfaces.ErrOperatorNotFound
	}
	return nil
}

func (r *OperatorRepository) Delete(ctx context.Context, operatorID string) error {
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM operators WHERE operator_id::text = $1
	`, operatorID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return interfaces.ErrOperatorNotFound
	}
	return nil
}
