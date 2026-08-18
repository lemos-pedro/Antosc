package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
)

type BackhaulInterfaceService struct {
	repo interfaces.BackhaulInterfaceRepository
}

func NewBackhaulInterfaceService(repo interfaces.BackhaulInterfaceRepository) *BackhaulInterfaceService {
	return &BackhaulInterfaceService{repo: repo}
}

func (s *BackhaulInterfaceService) CollectInterface(ctx context.Context, iface *domain.BackhaulInterface) error {
	if iface == nil {
		return errors.New("interface cannot be nil")
	}
	if iface.TowerID == uuid.Nil || iface.Name == "" || iface.AdminStatus == "" || iface.OperStatus == "" {
		return errors.New("tower_id, interface_name, admin_status and oper_status are required")
	}
	if iface.MeasuredAt.IsZero() {
		iface.MeasuredAt = time.Now().UTC()
	}
	if iface.MeasuredAt.After(time.Now().Add(5 * time.Minute)) {
		return errors.New("measured_at cannot be in the future")
	}
	if err := s.repo.Create(ctx, iface); err != nil {
		return fmt.Errorf("create backhaul measurement: %w", err)
	}
	return nil
}

func (s *BackhaulInterfaceService) CollectMultipleInterfaces(ctx context.Context, ifaces []*domain.BackhaulInterface) error {
	if len(ifaces) == 0 {
		return nil
	}
	for _, iface := range ifaces {
		if iface == nil {
			return errors.New("backhaul batch contains a nil interface")
		}
		if iface.TowerID == uuid.Nil || iface.Name == "" || iface.AdminStatus == "" || iface.OperStatus == "" {
			return errors.New("each backhaul measurement needs tower_id, interface_name, admin_status and oper_status")
		}
		if iface.MeasuredAt.IsZero() {
			iface.MeasuredAt = time.Now().UTC()
		}
	}
	if err := s.repo.CreateMany(ctx, ifaces); err != nil {
		return fmt.Errorf("create backhaul batch: %w", err)
	}
	return nil
}

func (s *BackhaulInterfaceService) GetLatestInterface(ctx context.Context, towerID uuid.UUID, interfaceName string) (*domain.BackhaulInterface, error) {
	if towerID == uuid.Nil || interfaceName == "" {
		return nil, errors.New("tower_id and interface_name are required")
	}
	return s.repo.GetLatest(ctx, towerID, interfaceName)
}

func (s *BackhaulInterfaceService) GetInterfaceHistory(ctx context.Context, towerID uuid.UUID, interfaceName string, limit, offset int, after, before *time.Time) ([]*domain.BackhaulInterface, int, error) {
	return s.List(ctx, &domain.BackhaulInterfaceFilter{TowerID: towerID, InterfaceName: &interfaceName, Limit: limit, Offset: offset, OrderBy: []string{"measured_at DESC"}, MeasuredAtAfter: after, MeasuredAtBefore: before})
}

func (s *BackhaulInterfaceService) List(ctx context.Context, filter *domain.BackhaulInterfaceFilter) ([]*domain.BackhaulInterface, int, error) {
	if filter == nil || filter.TowerID == uuid.Nil {
		return nil, 0, errors.New("tower_id is required")
	}
	return s.repo.List(ctx, filter)
}

// GetTowerBackhaulStatus returns the newest record for every interface.
func (s *BackhaulInterfaceService) GetTowerBackhaulStatus(ctx context.Context, towerID uuid.UUID) ([]*domain.BackhaulInterface, error) {
	all, _, err := s.List(ctx, &domain.BackhaulInterfaceFilter{TowerID: towerID, Limit: 1000, OrderBy: []string{"measured_at DESC"}})
	if err != nil {
		return nil, fmt.Errorf("get tower backhaul status: %w", err)
	}
	seen := make(map[string]struct{}, len(all))
	latest := make([]*domain.BackhaulInterface, 0, len(all))
	for _, iface := range all {
		if _, ok := seen[iface.Name]; ok {
			continue
		}
		seen[iface.Name] = struct{}{}
		latest = append(latest, iface)
	}
	return latest, nil
}
