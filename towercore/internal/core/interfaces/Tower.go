package interfaces

import (
	"context"
	"errors"

	"towercore/internal/core/domain"
)

var ErrTowerNotFound = errors.New("tower not found")

// TowerFilter define os filtros aceites na listagem de torres.
type TowerFilter struct {
	Status     string
	OperatorID string
	RegionID   string
	Limit      int
	Offset     int
}

type TowerCache interface {
	GetList(filter TowerFilter) ([]domain.Tower, int, bool)
	SetList(filter TowerFilter, towers []domain.Tower, total int)
	GetByID(id string) (*domain.Tower, bool)
	SetByID(tower *domain.Tower)
	InvalidateTower(id string)
	InvalidateList()
}

// TowerRepository define o contrato de persistência para torres.
type TowerRepository interface {
	List(ctx context.Context, filter TowerFilter) ([]domain.Tower, int, error)
	GetByID(ctx context.Context, id string) (*domain.Tower, error)
	Upsert(ctx context.Context, tower *domain.Tower) error
}
