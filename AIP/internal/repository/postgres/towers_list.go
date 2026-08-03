package postgres

import (
	"context"
	"database/sql"
)

// ListAll devolve todas as torres da cache local — dimensão para Power BI.
func (r *towerRepository) ListAll(ctx context.Context) ([]Tower, error) {
	rows, err := r.db.DB.QueryContext(ctx, `
		SELECT tower_id, name, vendor, operator_id, region_id,
			availability_7d, availability_30d
		FROM towers
		ORDER BY tower_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Tower
	for rows.Next() {
		var t Tower
		var avail7, avail30 sql.NullFloat64
		if err := rows.Scan(
			&t.TowerID, &t.Name, &t.Vendor, &t.OperatorID, &t.RegionID,
			&avail7, &avail30,
		); err != nil {
			return nil, err
		}
		t.Availability7d = avail7
		t.Availability30d = avail30
		out = append(out, t)
	}
	return out, rows.Err()
}
