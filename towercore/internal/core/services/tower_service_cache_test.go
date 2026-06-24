package services

import (
	"context"
	"testing"
	"time"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
	"towercore/internal/infrastructure/cache"
)

type countingTowerRepo struct {
	listCalls   int
	getCalls    int
	upsertCalls int
	towers      []domain.Tower
}

func newCountingTowerRepo() *countingTowerRepo {
	now := time.Now().UTC()
	return &countingTowerRepo{
		towers: []domain.Tower{
			{
				ID: "tower-1", Name: "Tower 1", Status: domain.TowerStatusOnline, UpdatedAt: now, CreatedAt: now,
			},
		},
	}
}

func (r *countingTowerRepo) List(_ context.Context, filter interfaces.TowerFilter) ([]domain.Tower, int, error) {
	r.listCalls++
	return append([]domain.Tower(nil), r.towers...), len(r.towers), nil
}

func (r *countingTowerRepo) GetByID(_ context.Context, id string) (*domain.Tower, error) {
	r.getCalls++
	for _, tower := range r.towers {
		if tower.ID == id {
			copy := tower
			return &copy, nil
		}
	}
	return nil, interfaces.ErrTowerNotFound
}

func (r *countingTowerRepo) Upsert(_ context.Context, tower *domain.Tower) error {
	r.upsertCalls++
	for i := range r.towers {
		if r.towers[i].ID == tower.ID {
			r.towers[i] = *tower
			return nil
		}
	}
	r.towers = append(r.towers, *tower)
	return nil
}

func TestTowerService_UsesCacheForListAndDetail(t *testing.T) {
	repo := newCountingTowerRepo()
	towerCache := cache.NewTowerCache(time.Minute, time.Minute)
	svc := NewTowerServiceWithCache(repo, towerCache)

	filter := interfaces.TowerFilter{Status: "online", Limit: 50, Offset: 0}

	if _, _, err := svc.List(context.Background(), filter); err != nil {
		t.Fatalf("first list failed: %v", err)
	}
	if _, _, err := svc.List(context.Background(), filter); err != nil {
		t.Fatalf("second list failed: %v", err)
	}
	if repo.listCalls != 1 {
		t.Fatalf("expected 1 repo list call, got %d", repo.listCalls)
	}

	if _, err := svc.GetByID(context.Background(), "tower-1"); err != nil {
		t.Fatalf("first get failed: %v", err)
	}
	if _, err := svc.GetByID(context.Background(), "tower-1"); err != nil {
		t.Fatalf("second get failed: %v", err)
	}
	if repo.getCalls != 0 {
		t.Fatalf("expected detail to be served from list-warmed cache, got %d repo get calls", repo.getCalls)
	}
}

func TestTowerService_InvalidatesCacheOnSave(t *testing.T) {
	repo := newCountingTowerRepo()
	towerCache := cache.NewTowerCache(time.Minute, time.Minute)
	svc := NewTowerServiceWithCache(repo, towerCache)

	filter := interfaces.TowerFilter{Status: "online", Limit: 50, Offset: 0}
	if _, _, err := svc.List(context.Background(), filter); err != nil {
		t.Fatalf("initial list failed: %v", err)
	}

	tower, err := svc.GetByID(context.Background(), "tower-1")
	if err != nil {
		t.Fatalf("initial get failed: %v", err)
	}
	tower.Name = "Tower 1 Updated"

	if err := svc.Save(context.Background(), tower); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	if _, _, err := svc.List(context.Background(), filter); err != nil {
		t.Fatalf("list after save failed: %v", err)
	}
	if repo.listCalls != 2 {
		t.Fatalf("expected list cache invalidation after save, got %d repo list calls", repo.listCalls)
	}
}
