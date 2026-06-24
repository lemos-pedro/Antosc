package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
)

// DiscoveredDevicePromotionService orquestra a transição de um
// DiscoveredDevice (em triagem) para uma Tower real e gerida pelo
// sistema. Não decide políticas de negócio sobre operador/região —
// isso continua a ser uma decisão humana, fornecida no momento da
// promoção (normalmente via formulário na UI ou payload da API).
type DiscoveredDevicePromotionService struct {
	deviceRepo interfaces.DiscoveredDeviceRepository
	towerSvc   *TowerService
	now        func() time.Time
}

func NewDiscoveredDevicePromotionService(
	deviceRepo interfaces.DiscoveredDeviceRepository,
	towerSvc *TowerService,
) *DiscoveredDevicePromotionService {
	return &DiscoveredDevicePromotionService{
		deviceRepo: deviceRepo,
		towerSvc:   towerSvc,
		now:        time.Now,
	}
}

// PromotionInput agrupa os dados que só um humano pode fornecer ao
// promover um dispositivo descoberto para Tower.
type PromotionInput struct {
	Name       string
	OperatorID string
	RegionID   string
	// Vendor é opcional: se vazio, usa-se o DetectedVendor do
	// dispositivo (identificado automaticamente via sysObjectID).
	Vendor string
}

// Promote cria uma Tower a partir de um DiscoveredDevice e marca o
// dispositivo como 'promoted'. O IP, a community e a versão SNMP já
// detetados pelo discovery são herdados automaticamente.
func (s *DiscoveredDevicePromotionService) Promote(
	ctx context.Context,
	deviceID string,
	input PromotionInput,
) (*domain.Tower, error) {
	device, err := s.deviceRepo.GetByID(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("buscar dispositivo descoberto: %w", err)
	}
	if device.Status == domain.DiscoveredDeviceStatusPromoted {
		return nil, fmt.Errorf("dispositivo %s já foi promovido", deviceID)
	}

	vendor := strings.ToLower(strings.TrimSpace(input.Vendor))
	if vendor == "" {
		vendor = device.DetectedVendor
	}
	if vendor == "" {
		return nil, fmt.Errorf(
			"vendor não identificado automaticamente para %s — informa o vendor manualmente na promoção",
			device.IPAddress,
		)
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = fmt.Sprintf("Tower-%s", device.IPAddress)
	}

	tower := &domain.Tower{
		Name:          name,
		Status:        domain.TowerStatusOffline, // confirma-se como online no primeiro polling bem-sucedido
		OperatorID:    strings.TrimSpace(input.OperatorID),
		RegionID:      strings.TrimSpace(input.RegionID),
		Vendor:        vendor,
		SNMPEnabled:   true,
		SNMPVersion:   device.SNMPVersion,
		SNMPTarget:    device.IPAddress,
		SNMPCommunity: device.SNMPCommunity,
		CreatedAt:     s.now().UTC(),
	}

	if err := s.towerSvc.Save(ctx, tower); err != nil {
		return nil, fmt.Errorf("criar tower a partir do dispositivo descoberto: %w", err)
	}

	if err := s.deviceRepo.UpdateStatus(ctx, deviceID, domain.DiscoveredDeviceStatusPromoted, tower.ID); err != nil {
		return nil, fmt.Errorf("marcar dispositivo como promovido: %w", err)
	}

	return tower, nil
}

// Ignore marca um dispositivo descoberto como ignorado — usado quando
// o operador decide que aquele IP não corresponde a uma torre a gerir
// (ex: outro equipamento de rede que respondeu por acidente ao scan).
func (s *DiscoveredDevicePromotionService) Ignore(ctx context.Context, deviceID string) error {
	return s.deviceRepo.UpdateStatus(ctx, deviceID, domain.DiscoveredDeviceStatusIgnored, "")
}
