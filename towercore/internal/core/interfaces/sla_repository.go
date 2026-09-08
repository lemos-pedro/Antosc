package interfaces

import (
	"context"
	"errors"

	"towercore/internal/core/domain"
)

type SLARepository interface {
	GetGlobal(ctx context.Context) (*domain.SLA, error)
	// GetByRegion devolve o mesmo cálculo de GetGlobal mas filtrado a uma
	// única região. Devolve ErrRegionNotFound se region_id não existir.
	GetByRegion(ctx context.Context, regionID string) (*domain.SLA, error)
}

var ErrRegionNotFound = errors.New("region not found")
