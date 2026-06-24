package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
)

// SNMPInventoryEntry describes one tower credential entry loaded from config.
type SNMPInventoryEntry struct {
	ID              string   `json:"tower_id"`
	Name            string   `json:"name"`
	Status          string   `json:"status"`
	OperatorID      string   `json:"operator_id"`
	RegionID        string   `json:"region_id"`
	Vendor          string   `json:"vendor"`
	SNMPEnabled     *bool    `json:"snmp_enabled"`
	SNMPVersion     string   `json:"snmp_version"`
	SNMPTarget      string   `json:"snmp_target"`
	SNMPCommunity   string   `json:"snmp_community"`
	SNMPV3User      string   `json:"snmp_v3_user"`
	SNMPAuthProto   string   `json:"snmp_auth_protocol"`
	SNMPAuthPass    string   `json:"snmp_auth_password"`
	SNMPPrivProto   string   `json:"snmp_priv_protocol"`
	SNMPPrivPass    string   `json:"snmp_priv_password"`
	Availability30d *float64 `json:"availability_30d"`
}

// ApplySNMPInventory upserts SNMP connectivity details from a JSON inventory.
func (s *TowerService) ApplySNMPInventory(ctx context.Context, raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}

	entries, err := parseSNMPInventory(raw)
	if err != nil {
		return 0, err
	}

	applied := 0
	for i, entry := range entries {
		tower, err := s.towerFromInventory(ctx, entry)
		if err != nil {
			return applied, fmt.Errorf("snmp inventory entry %d: %w", i, err)
		}
		if err := s.Save(ctx, tower); err != nil {
			return applied, fmt.Errorf("snmp inventory entry %d: %w", i, err)
		}
		applied++
	}
	return applied, nil
}

func parseSNMPInventory(raw string) ([]SNMPInventoryEntry, error) {
	var entries []SNMPInventoryEntry
	if err := json.Unmarshal([]byte(raw), &entries); err == nil {
		return entries, nil
	}

	var payload struct {
		Towers []SNMPInventoryEntry `json:"towers"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, fmt.Errorf("SNMP_TOWERS_JSON must be a JSON array or an object with a towers array: %w", err)
	}
	return payload.Towers, nil
}

func (s *TowerService) towerFromInventory(ctx context.Context, entry SNMPInventoryEntry) (*domain.Tower, error) {
	id := strings.TrimSpace(entry.ID)
	var tower *domain.Tower

	if id != "" {
		existing, err := s.repo.GetByID(ctx, id)
		switch {
		case err == nil:
			tower = existing
		case errors.Is(err, interfaces.ErrTowerNotFound):
			tower = &domain.Tower{ID: id}
		default:
			return nil, err
		}
	} else {
		tower = &domain.Tower{}
	}

	applyInventoryFields(tower, entry)
	if strings.TrimSpace(tower.Name) == "" {
		return nil, errors.New("name is required for new tower inventory entries")
	}
	return tower, nil
}

func applyInventoryFields(tower *domain.Tower, entry SNMPInventoryEntry) {
	if v := strings.TrimSpace(entry.Name); v != "" {
		tower.Name = v
	}
	if v := strings.TrimSpace(entry.Status); v != "" {
		tower.Status = domain.TowerStatus(v)
	}
	if v := strings.TrimSpace(entry.OperatorID); v != "" {
		tower.OperatorID = v
	}
	if v := strings.TrimSpace(entry.RegionID); v != "" {
		tower.RegionID = v
	}
	if v := strings.TrimSpace(entry.Vendor); v != "" {
		tower.Vendor = v
	}
	if entry.SNMPEnabled != nil {
		tower.SNMPEnabled = *entry.SNMPEnabled
	} else {
		tower.SNMPEnabled = true
	}
	if v := strings.TrimSpace(entry.SNMPVersion); v != "" {
		tower.SNMPVersion = v
	}
	if v := strings.TrimSpace(entry.SNMPTarget); v != "" {
		tower.SNMPTarget = v
	}
	if v := strings.TrimSpace(entry.SNMPCommunity); v != "" {
		tower.SNMPCommunity = v
	}
	if v := strings.TrimSpace(entry.SNMPV3User); v != "" {
		tower.SNMPV3User = v
	}
	if v := strings.TrimSpace(entry.SNMPAuthProto); v != "" {
		tower.SNMPAuthProto = v
	}
	if v := strings.TrimSpace(entry.SNMPAuthPass); v != "" {
		tower.SNMPAuthPass = v
	}
	if v := strings.TrimSpace(entry.SNMPPrivProto); v != "" {
		tower.SNMPPrivProto = v
	}
	if v := strings.TrimSpace(entry.SNMPPrivPass); v != "" {
		tower.SNMPPrivPass = v
	}
	if entry.Availability30d != nil {
		tower.Availability30d = *entry.Availability30d
	}
}
