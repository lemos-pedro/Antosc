package mock

import (
	"context"
	"slices"
	"sync"
	"time"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
)

type TowerRepository struct {
	mu     sync.RWMutex
	towers []domain.Tower
}

func NewTowerRepository() *TowerRepository {
	now := time.Now().UTC()
	return &TowerRepository{
		towers: []domain.Tower{
			{
				ID:              "00000000-0000-0000-0000-000000000001",
				Name:            "Tower-001",
				Status:          domain.TowerStatusOnline,
				Vendor:          "eltek",
				SNMPEnabled:     true,
				SNMPTarget:      "10.0.0.11",
				SNMPCommunity:   "Antosc-noc",
				OperatorID:      "00000000-0000-0000-0000-000000000101",
				RegionID:        "00000000-0000-0000-0000-000000000201",
				Availability30d: 99.97,
				UpdatedAt:       now,
				CreatedAt:       now,
			},
			{
				ID:              "00000000-0000-0000-0000-000000000002",
				Name:            "Tower-002",
				Status:          domain.TowerStatusDegraded,
				Vendor:          "huawei",
				SNMPEnabled:     true,
				SNMPTarget:      "10.10.0.12",
				SNMPCommunity:   "Antosc-noc",
				OperatorID:      "00000000-0000-0000-0000-000000000102",
				RegionID:        "00000000-0000-0000-0000-000000000201",
				Availability30d: 98.10,
				UpdatedAt:       now,
				CreatedAt:       now,
			},
			{
				ID:              "00000000-0000-0000-0000-000000000003",
				Name:            "Tower-003",
				Status:          domain.TowerStatusOffline,
				Vendor:          "enetek",
				SNMPEnabled:     false,
				SNMPTarget:      "10.10.0.13",
				SNMPCommunity:   "Antosc-noc",
				OperatorID:      "00000000-0000-0000-0000-000000000101",
				RegionID:        "00000000-0000-0000-0000-000000000202",
				Availability30d: 93.42,
				UpdatedAt:       now,
				CreatedAt:       now,
			},
		},
	}
}

func (r *TowerRepository) List(_ context.Context, filter interfaces.TowerFilter) ([]domain.Tower, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	filtered := make([]domain.Tower, 0, len(r.towers))
	for _, tower := range r.towers {
		if filter.Status != "" && string(tower.Status) != filter.Status {
			continue
		}
		if filter.OperatorID != "" && tower.OperatorID != filter.OperatorID {
			continue
		}
		if filter.RegionID != "" && tower.RegionID != filter.RegionID {
			continue
		}
		filtered = append(filtered, tower)
	}

	total := len(filtered)
	start := min(filter.Offset, total)
	end := min(start+filter.Limit, total)

	return slices.Clone(filtered[start:end]), total, nil
}

func (r *TowerRepository) GetByID(_ context.Context, id string) (*domain.Tower, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, tower := range r.towers {
		if tower.ID == id {
			c := tower
			return &c, nil
		}
	}
	return nil, interfaces.ErrTowerNotFound
}

func (r *TowerRepository) Upsert(_ context.Context, tower *domain.Tower) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.towers {
		if r.towers[i].ID == tower.ID {
			r.towers[i] = *tower
			return nil
		}
	}

	r.towers = append(r.towers, *tower)
	return nil
}
