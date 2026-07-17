package services

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
)

type TowerService struct {
	repo      interfaces.TowerRepository
	cache     interfaces.TowerCache
	auditRepo interfaces.AuditRepository
}

func NewTowerService(repo interfaces.TowerRepository, auditRepo ...interfaces.AuditRepository) *TowerService {
	return NewTowerServiceWithCache(repo, nil, auditRepo...)
}

func NewTowerServiceWithCache(repo interfaces.TowerRepository, cache interfaces.TowerCache, auditRepo ...interfaces.AuditRepository) *TowerService {
	var ar interfaces.AuditRepository
	if len(auditRepo) > 0 {
		ar = auditRepo[0]
	}
	return &TowerService{repo: repo, cache: cache, auditRepo: ar}
}

func (s *TowerService) List(ctx context.Context, filter interfaces.TowerFilter) ([]domain.Tower, int, error) {
	if filter.Limit <= 0 || filter.Limit > 2000 {
		filter.Limit = 500
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	if s.cache != nil {
		if towers, total, ok := s.cache.GetList(filter); ok {
			return towers, total, nil
		}
	}

	towers, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	if s.cache != nil {
		s.cache.SetList(filter, towers, total)
		for i := range towers {
			tower := towers[i]
			s.cache.SetByID(&tower)
		}
	}
	return towers, total, nil
}

func (s *TowerService) GetByID(ctx context.Context, id string) (*domain.Tower, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("tower_id is required")
	}
	if s.cache != nil {
		if tower, ok := s.cache.GetByID(id); ok {
			return tower, nil
		}
	}

	tower, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if s.cache != nil {
		s.cache.SetByID(tower)
	}
	return tower, nil
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
		if err := validateSNMPTarget(tower.SNMPTarget); err != nil {
			return err
		}
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

	if err := s.repo.Upsert(ctx, tower); err != nil {
		return err
	}
	s.invalidateTowerCache(tower.ID)
	return nil
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
		if err := validateSNMPTarget(tower.SNMPTarget); err != nil {
			return nil, err
		}
		if err := validateSNMPCredentials(tower); err != nil {
			return nil, err
		}
	}

	tower.UpdatedAt = time.Now().UTC()
	if err := s.repo.Upsert(ctx, tower); err != nil {
		return nil, err
	}
	s.invalidateTowerCache(tower.ID)
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

func (s *TowerService) invalidateTowerCache(towerID string) {
	if s.cache == nil {
		return
	}
	s.cache.InvalidateTower(towerID)
	s.cache.InvalidateList()
}

func (s *TowerService) UpdateStatus(ctx context.Context, towerID string, status domain.TowerStatus) error {
	towerID = strings.TrimSpace(towerID)
	if towerID == "" {
		return errors.New("tower_id is required")
	}

	switch status {
	case domain.TowerStatusOnline, domain.TowerStatusDegraded, domain.TowerStatusOffline:
	default:
		return errors.New("invalid status")
	}

	tower, err := s.repo.GetByID(ctx, towerID)
	if err != nil {
		return err
	}

	if tower.Status == status {
		return nil
	}

	tower.Status = status
	tower.UpdatedAt = time.Now().UTC()

	if err := s.repo.Upsert(ctx, tower); err != nil {
		return err
	}
	s.invalidateTowerCache(tower.ID)
	return nil
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

func validateSNMPTarget(target string) error {
	hostPort := strings.TrimSpace(target)
	if hostPort == "" {
		return errors.New("snmp_target is required when snmp_enabled=true")
	}

	if strings.Contains(hostPort, "://") {
		parsed, err := url.Parse(hostPort)
		if err != nil {
			return errors.New("snmp_target must be a valid IP, hostname, or udp://host[:port]")
		}
		switch strings.ToLower(parsed.Scheme) {
		case "udp", "snmp":
		default:
			return errors.New("snmp_target scheme must be udp or snmp")
		}
		hostPort = parsed.Host
	}

	host := hostPort
	if h, p, err := net.SplitHostPort(hostPort); err == nil {
		host = h
		if err := validateSNMPPort(p); err != nil {
			return err
		}
	} else if strings.Contains(err.Error(), "too many colons") {
		if net.ParseIP(hostPort) == nil {
			return errors.New("snmp_target must be a valid IP, hostname, or host:port")
		}
	}

	host = strings.Trim(host, "[]")
	if host == "" {
		return errors.New("snmp_target host is required")
	}
	if strings.ContainsAny(host, "/ \t\r\n") {
		return errors.New("snmp_target must be a valid IP, hostname, or host:port")
	}
	return nil
}

func validateSNMPPort(raw string) error {
	port, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || port < 1 || port > 65535 {
		return errors.New("snmp_target port must be between 1 and 65535")
	}
	return nil
}

// ListOperators devolve os operadores associados a uma torre.
func (s *TowerService) ListOperators(ctx context.Context, towerID string) ([]domain.Operator, error) {
	towerID = strings.TrimSpace(towerID)
	if towerID == "" {
		return nil, errors.New("tower_id is required")
	}
	return s.repo.ListOperators(ctx, towerID)
}

// AddOperator associa um operador a uma torre (relação N:N via site_operators).
func (s *TowerService) AddOperator(ctx context.Context, towerID, operatorID string) error {
	towerID = strings.TrimSpace(towerID)
	operatorID = strings.TrimSpace(operatorID)
	if towerID == "" {
		return errors.New("tower_id is required")
	}
	if operatorID == "" {
		return errors.New("operator_id is required")
	}

	if _, err := s.repo.GetByID(ctx, towerID); err != nil {
		return err
	}

	if err := s.repo.AddOperator(ctx, towerID, operatorID); err != nil {
		return err
	}
	s.invalidateTowerCache(towerID)
	return nil
}

// RemoveOperator remove a associação entre torre e operador.
func (s *TowerService) RemoveOperator(ctx context.Context, towerID, operatorID string) error {
	towerID = strings.TrimSpace(towerID)
	operatorID = strings.TrimSpace(operatorID)
	if towerID == "" {
		return errors.New("tower_id is required")
	}
	if operatorID == "" {
		return errors.New("operator_id is required")
	}

	if err := s.repo.RemoveOperator(ctx, towerID, operatorID); err != nil {
		return err
	}
	s.invalidateTowerCache(towerID)
	return nil
}