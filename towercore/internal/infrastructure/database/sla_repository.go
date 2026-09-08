package database

import (
	"context"
	"database/sql"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
)

type SLARepository struct {
	db *sql.DB
}

func NewSLARepository(db *sql.DB) *SLARepository {
	return &SLARepository{db: db}
}

func (r *SLARepository) GetGlobal(ctx context.Context) (*domain.SLA, error) {
	var sla domain.SLA

	err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) AS total_sites,

			COUNT(*) FILTER (WHERE status = 'online') AS online_sites,
			COUNT(*) FILTER (WHERE status = 'offline') AS offline_sites,
			COUNT(*) FILTER (WHERE status = 'degraded') AS degraded_sites,

			ROUND(
				(
					COUNT(*) FILTER (WHERE status IN ('online','degraded'))::numeric
					/
					NULLIF(COUNT(*),0)
				) * 100,
			2)
		FROM towers
	`).Scan(
		&sla.TotalSites,
		&sla.OnlineSites,
		&sla.OfflineSites,
		&sla.DegradedSites,
		&sla.Availability,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return &domain.SLA{}, nil
		}
		return nil, err
	}

	return &sla, nil
}

// GetByRegion aplica o mesmo cálculo de GetGlobal filtrado por região, e
// inclui o nome da região no resultado (join simples com regions).
func (r *SLARepository) GetByRegion(ctx context.Context, regionID string) (*domain.SLA, error) {
	var regionName string
	if err := r.db.QueryRowContext(ctx,
		`SELECT name FROM regions WHERE region_id::text = $1`, regionID,
	).Scan(&regionName); err != nil {
		if err == sql.ErrNoRows {
			return nil, interfaces.ErrRegionNotFound
		}
		return nil, err
	}

	sla := domain.SLA{RegionID: regionID, RegionName: regionName}

	err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) AS total_sites,

			COUNT(*) FILTER (WHERE status = 'online') AS online_sites,
			COUNT(*) FILTER (WHERE status = 'offline') AS offline_sites,
			COUNT(*) FILTER (WHERE status = 'degraded') AS degraded_sites,

			ROUND(
				(
					COUNT(*) FILTER (WHERE status IN ('online','degraded'))::numeric
					/
					NULLIF(COUNT(*),0)
				) * 100,
			2)
		FROM towers
		WHERE region_id::text = $1
	`, regionID).Scan(
		&sla.TotalSites,
		&sla.OnlineSites,
		&sla.OfflineSites,
		&sla.DegradedSites,
		&sla.Availability,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return &sla, nil
		}
		return nil, err
	}

	return &sla, nil
}
