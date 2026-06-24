package cache

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
)

type TowerCache struct {
	mu            sync.RWMutex
	listTTL       time.Duration
	detailTTL     time.Duration
	listEntries   map[string]listEntry
	detailEntries map[string]detailEntry
}

type listEntry struct {
	towers    []domain.Tower
	total     int
	expiresAt time.Time
}

type detailEntry struct {
	tower     domain.Tower
	expiresAt time.Time
}

func NewTowerCache(listTTL, detailTTL time.Duration) *TowerCache {
	return &TowerCache{
		listTTL:       listTTL,
		detailTTL:     detailTTL,
		listEntries:   make(map[string]listEntry),
		detailEntries: make(map[string]detailEntry),
	}
}

func (c *TowerCache) GetList(filter interfaces.TowerFilter) ([]domain.Tower, int, bool) {
	if c == nil || c.listTTL <= 0 {
		return nil, 0, false
	}

	key := listKey(filter)
	now := time.Now().UTC()

	c.mu.RLock()
	entry, ok := c.listEntries[key]
	c.mu.RUnlock()
	if !ok || now.After(entry.expiresAt) {
		if ok {
			c.mu.Lock()
			delete(c.listEntries, key)
			c.mu.Unlock()
		}
		return nil, 0, false
	}

	return cloneTowers(entry.towers), entry.total, true
}

func (c *TowerCache) SetList(filter interfaces.TowerFilter, towers []domain.Tower, total int) {
	if c == nil || c.listTTL <= 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.listEntries[listKey(filter)] = listEntry{
		towers:    cloneTowers(towers),
		total:     total,
		expiresAt: time.Now().UTC().Add(c.listTTL),
	}
}

func (c *TowerCache) GetByID(id string) (*domain.Tower, bool) {
	if c == nil || c.detailTTL <= 0 {
		return nil, false
	}

	id = strings.TrimSpace(id)
	now := time.Now().UTC()

	c.mu.RLock()
	entry, ok := c.detailEntries[id]
	c.mu.RUnlock()
	if !ok || now.After(entry.expiresAt) {
		if ok {
			c.mu.Lock()
			delete(c.detailEntries, id)
			c.mu.Unlock()
		}
		return nil, false
	}

	tower := cloneTower(entry.tower)
	return &tower, true
}

func (c *TowerCache) SetByID(tower *domain.Tower) {
	if c == nil || c.detailTTL <= 0 || tower == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.detailEntries[strings.TrimSpace(tower.ID)] = detailEntry{
		tower:     cloneTower(*tower),
		expiresAt: time.Now().UTC().Add(c.detailTTL),
	}
}

func (c *TowerCache) InvalidateTower(id string) {
	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.detailEntries, strings.TrimSpace(id))
}

func (c *TowerCache) InvalidateList() {
	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	clear(c.listEntries)
}

func listKey(filter interfaces.TowerFilter) string {
	return fmt.Sprintf(
		"status=%s|operator=%s|region=%s|limit=%d|offset=%d",
		strings.TrimSpace(filter.Status),
		strings.TrimSpace(filter.OperatorID),
		strings.TrimSpace(filter.RegionID),
		filter.Limit,
		filter.Offset,
	)
}

func cloneTowers(in []domain.Tower) []domain.Tower {
	return slices.Clone(in)
}

func cloneTower(in domain.Tower) domain.Tower {
	return in
}
