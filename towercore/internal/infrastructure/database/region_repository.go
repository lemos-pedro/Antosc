package database

import (
	"context"
	"database/sql"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
)

type RegionRepository struct {
	db *sql.DB
}

func NewRegionRepository(db *sql.DB) *RegionRepository {
	return &RegionRepository{db: db}
}

func (r *RegionRepository) List(ctx context.Context) ([]domain.Region, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT region_id, name, created_at
		FROM regions
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var regions []domain.Region

	for rows.Next() {
		var region domain.Region

		if err := rows.Scan(
			&region.RegionID,
			&region.Name,
			&region.CreatedAt,
		); err != nil {
			return nil, err
		}

		regions = append(regions, region)
	}

	return regions, nil
}

func (r *RegionRepository) Create(ctx context.Context, region *domain.Region) error {
	return r.db.QueryRowContext(
		ctx,
		`
		INSERT INTO regions (name)
		VALUES ($1)
		RETURNING region_id, created_at
		`,
		region.Name,
	).Scan(
		&region.RegionID,
		&region.CreatedAt,
	)
}

func (r *RegionRepository) GetByID(ctx context.Context, regionID string) (*domain.Region, error) {
	var region domain.Region
	err := r.db.QueryRowContext(ctx, `
		SELECT region_id, name, created_at
		FROM regions
		WHERE region_id::text = $1
	`, regionID).Scan(
		&region.RegionID,
		&region.Name,
		&region.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, interfaces.ErrRegionNotFound
		}
		return nil, err
	}
	return &region, nil
}
