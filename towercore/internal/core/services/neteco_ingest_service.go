package services

import (
	"log"

	"towercore/internal/core/interfaces"
)

type NetEcoIngestService struct {
	client      interfaces.NetEcoClient
	towerRepo   interfaces.TowerRepository   // já deves ter isto no projeto
	eventRepo   interfaces.EventRepository   // idem
}

func NewNetEcoIngestService(client interfaces.NetEcoClient, towerRepo interfaces.TowerRepository, eventRepo interfaces.EventRepository) *NetEcoIngestService {
	return &NetEcoIngestService{
		client:    client,
		towerRepo: towerRepo,
		eventRepo: eventRepo,
	}
}

// siteDnByTowerName mapeia nome da torre no towercore -> siteDn no NetEco.
// Idealmente isto vive numa coluna nova em `towers` (ex: neteco_site_dn),
// não hardcoded — placeholder até termos essa migration.
func (s *NetEcoIngestService) PollBatteryStatus(siteDnByTowerName map[string]string) {
	for towerName, siteDn := range siteDnByTowerName {
		status, err := s.client.FetchBatteryStatus(siteDn)
		if err != nil {
			log.Printf("[NetEco] erro ao consultar %s (%s): %v", towerName, siteDn, err)
			continue
		}

		tower, err := s.towerRepo.FindByName(towerName)
		if err != nil {
			log.Printf("[NetEco] torre não encontrada: %s", towerName)
			continue
		}

		s.applyBatteryStatus(tower, status)
	}
}

func (s *NetEcoIngestService) applyBatteryStatus(tower interfaces.Tower, status *interfaces.BatteryStatus) {
	// Nunca fabricar dados: -1 do NetEco vira nil no domínio
	if status.SOC >= 0 {
		soc := status.SOC
		tower.BatterySOC = &soc
	} else {
		tower.BatterySOC = nil
	}
	if status.SOH >= 0 {
		soh := status.SOH
		tower.BatterySOH = &soh
	} else {
		tower.BatterySOH = nil
	}

	if err := s.towerRepo.Update(tower); err != nil {
		log.Printf("[NetEco] erro ao gravar torre %s: %v", tower.Name, err)
	}
}