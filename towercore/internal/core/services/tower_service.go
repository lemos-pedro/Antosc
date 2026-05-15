package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
)

type TowerService struct {
	repo      interfaces.TowerRepository
	auditRepo interfaces.AuditRepository
}

func NewTowerService(repo interfaces.TowerRepository, auditRepo ...interfaces.AuditRepository) *TowerService {
	var ar interfaces.AuditRepository
	if len(auditRepo) > 0 {
		ar = auditRepo[0]
	}
	return &TowerService{repo: repo, auditRepo: ar}
}

func (s *TowerService) List(ctx context.Context, filter interfaces.TowerFilter) ([]domain.Tower, int, error) {
	if filter.Limit <= 0 || filter.Limit > 200 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	return s.repo.List(ctx, filter)
}

func (s *TowerService) GetByID(ctx context.Context, id string) (*domain.Tower, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("tower_id is required")
	}
	return s.repo.GetByID(ctx, id)
}

func (s *TowerService) Save(ctx context.Context, tower *domain.Tower) error {
	if strings.TrimSpace(tower.Name) == "" {
		return errors.New("name is required")
	}

	switch tower.Status {
	case "":
		tower.Status = domain.TowerStatusOffline
	case domain.TowerStatusOnline, domain.TowerStatusDegraded, domain.TowerStatusOffline:
	default:
		return errors.New("invalid status")
	}

	tower.Vendor = strings.ToLower(strings.TrimSpace(tower.Vendor))
	tower.SNMPVersion = normalizeSNMPVersion(tower.SNMPVersion)
	tower.SNMPTarget = strings.TrimSpace(tower.SNMPTarget)
	tower.SNMPCommunity = strings.TrimSpace(tower.SNMPCommunity)
	tower.SNMPV3User = strings.TrimSpace(tower.SNMPV3User)
	tower.SNMPAuthProto = strings.ToLower(strings.TrimSpace(tower.SNMPAuthProto))
	tower.SNMPPrivProto = strings.ToLower(strings.TrimSpace(tower.SNMPPrivProto))
	tower.SNMPAuthPass = strings.TrimSpace(tower.SNMPAuthPass)
	tower.SNMPPrivPass = strings.TrimSpace(tower.SNMPPrivPass)
	if tower.SNMPEnabled && tower.Vendor == "" {
		return errors.New("vendor is required when snmp_enabled=true")
	}
	if tower.SNMPEnabled {
		if err := validateSNMPCredentials(tower); err != nil {
			return err
		}
	}

	if tower.ID == "" {
		id, err := newUUIDv4()
		if err != nil {
			return err
		}
		tower.ID = id
	}

	now := time.Now().UTC()
	if tower.CreatedAt.IsZero() {
		tower.CreatedAt = now
	}
	tower.UpdatedAt = now

	return s.repo.Upsert(ctx, tower)
}

func (s *TowerService) ConfigureSNMP(
	ctx context.Context,
	actor, towerID, vendor, version, target, community, v3User, authProto, authPass, privProto, privPass string,
	enabled bool,
) (*domain.Tower, error) {
	if strings.TrimSpace(towerID) == "" {
		return nil, errors.New("tower_id is required")
	}

	tower, err := s.repo.GetByID(ctx, towerID)
	if err != nil {
		return nil, err
	}

	tower.Vendor = strings.ToLower(strings.TrimSpace(vendor))
	tower.SNMPVersion = normalizeSNMPVersion(version)
	tower.SNMPTarget = strings.TrimSpace(target)
	tower.SNMPCommunity = strings.TrimSpace(community)
	tower.SNMPV3User = strings.TrimSpace(v3User)
	tower.SNMPAuthProto = strings.ToLower(strings.TrimSpace(authProto))
	tower.SNMPAuthPass = strings.TrimSpace(authPass)
	tower.SNMPPrivProto = strings.ToLower(strings.TrimSpace(privProto))
	tower.SNMPPrivPass = strings.TrimSpace(privPass)
	tower.SNMPEnabled = enabled

	if tower.SNMPEnabled {
		if tower.Vendor == "" {
			return nil, errors.New("vendor is required when snmp_enabled=true")
		}
		if tower.SNMPTarget == "" {
			return nil, errors.New("snmp_target is required when snmp_enabled=true")
		}
		if err := validateSNMPCredentials(tower); err != nil {
			return nil, err
		}
	}

	tower.UpdatedAt = time.Now().UTC()
	if err := s.repo.Upsert(ctx, tower); err != nil {
		return nil, err
	}
	if s.auditRepo != nil {
		entry := &domain.AuditLog{
			Actor:      safeActor(actor),
			Action:     "snmp.credentials.update",
			Resource:   "tower",
			ResourceID: tower.ID,
			Details:    "snmp credentials rotated/updated",
			CreatedAt:  time.Now().UTC(),
		}
		entryID, err := newUUIDv4()
		if err == nil {
			entry.ID = entryID
			_ = s.auditRepo.Create(ctx, entry)
		}
	}
	return tower, nil
}

func safeActor(actor string) string {
	actor = strings.TrimSpace(actor)
	if actor == "" {
		return "system"
	}
	return actor
}

func normalizeSNMPVersion(version string) string {
	v := strings.ToLower(strings.TrimSpace(version))
	if v == "" {
		return "v2c"
	}
	return v
}

func validateSNMPCredentials(tower *domain.Tower) error {
	switch tower.SNMPVersion {
	case "v2c":
		if tower.SNMPCommunity == "" {
			return errors.New("snmp_community is required for snmp_version=v2c")
		}
	case "v3":
		if tower.SNMPV3User == "" {
			return errors.New("snmp_v3_user is required for snmp_version=v3")
		}
		if tower.SNMPAuthProto == "" || tower.SNMPAuthPass == "" {
			return errors.New("snmp_auth_protocol and snmp_auth_password are required for snmp_version=v3")
		}
		if (tower.SNMPPrivProto == "") != (tower.SNMPPrivPass == "") {
			return errors.New("snmp_priv_protocol and snmp_priv_password must be provided together")
		}
	default:
		return errors.New("snmp_version must be v2c or v3")
	}
	return nil
}
