package services

import (
	"context"
	"log"
	"strings"
	"time"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
)

type NetEcoIngestService struct {
	client    interfaces.NetEcoClient
	towerRepo interfaces.TowerRepository
	metricSvc *MetricService
}

func NewNetEcoIngestService(client interfaces.NetEcoClient, towerRepo interfaces.TowerRepository, metricSvc *MetricService) *NetEcoIngestService {
	return &NetEcoIngestService{client: client, towerRepo: towerRepo, metricSvc: metricSvc}
}

// netEcoConnectState classifica siteConnectStatus do NetEco.
// Valores observados variam (Connected/Disconnected, 0/1, etc.).
type netEcoConnectState int

const (
	netEcoConnectUnknown netEcoConnectState = iota
	netEcoConnectUp
	netEcoConnectDown
)

// classifyNetEcoConnect interpreta o campo siteConnectStatus devolvido pela API.
func classifyNetEcoConnect(raw string) netEcoConnectState {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return netEcoConnectUnknown
	}

	// Desconectado / offline / down
	for _, neg := range []string{
		"disconnect", "disconnected", "offline", "down", "unreachable",
		"not connect", "not_connect", "no connect", "link down", "0",
	} {
		if s == neg || strings.Contains(s, neg) {
			// evitar falso positivo em "not disconnected"
			if strings.Contains(s, "not disconnect") {
				continue
			}
			return netEcoConnectDown
		}
	}

	// Conectado / online / up
	for _, pos := range []string{
		"connect", "connected", "online", "up", "normal", "1",
	} {
		if s == pos || strings.Contains(s, pos) {
			return netEcoConnectUp
		}
	}

	return netEcoConnectUnknown
}

// PollEnergyStatus percorre todas as torres com NetecoEnabled=true e atualiza
// bateria + energia DC/retificador a partir do NetEco.
//
// Regra de status (Prioridade 2 — sites Huawei via NetEco):
//
//	API falhou (bateria e energia)     → offline
//	API ok + siteConnectStatus DOWN    → offline  (controladora/site inacessível)
//	API ok + siteConnectStatus UP      → online   (exceto se já degraded por alarme)
//	API ok + siteConnectStatus unknown → online   (fallback; log de aviso)
//
// Não sobrescreve degraded → online: alarmes/traps activos mantêm degraded.
func (s *NetEcoIngestService) PollEnergyStatus(ctx context.Context) {
	enabled := true
	towers, _, err := s.towerRepo.List(ctx, interfaces.TowerFilter{
		NetecoEnabled: &enabled,
		Limit:         500,
	})
	if err != nil {
		log.Printf("[NetEco] erro ao listar torres: %v", err)
		return
	}

	for i := range towers {
		tower := &towers[i]
		if tower.NetecoNEID == "" {
			continue
		}
		collectedAt := time.Now().UTC()
		collected := false
		var collectionErr error
		connectState := netEcoConnectUnknown
		var connectRaw string

		battery, err := s.client.FetchBatteryStatus(tower.NetecoNEID)
		if err != nil {
			log.Printf("[NetEco] erro bateria %s (%s): %v", tower.Name, tower.NetecoNEID, err)
			collectionErr = err
		} else {
			s.applyBatteryStatus(tower, battery)
			collected = true
			connectRaw = battery.ConnectStatus
			connectState = classifyNetEcoConnect(battery.ConnectStatus)
		}

		energy, err := s.client.FetchSiteCounterInfo(tower.NetecoNEID)
		if err != nil {
			log.Printf("[NetEco] erro energia %s (%s): %v", tower.Name, tower.NetecoNEID, err)
			collectionErr = err
		} else {
			s.applyEnergyStatus(tower, energy)
			collected = true
		}

		tower.LastCollectedAt = &collectedAt

		if !collected {
			tower.CollectionStatus = domain.CollectionStatusFailed
			if collectionErr != nil {
				tower.LastCollectionError = collectionErr.Error()
			}
			tower.Status = domain.TowerStatusOffline
			log.Printf(
				"[NetEco] status=offline tower=%s neid=%s reason=collection_failed",
				tower.Name, tower.NetecoNEID,
			)
		} else {
			tower.CollectionStatus = domain.CollectionStatusActive
			tower.LastSuccessfulAt = &collectedAt
			tower.LastCollectionError = ""

			switch connectState {
			case netEcoConnectDown:
				// API NetEco respondeu, mas o site/controladora está desconectado.
				tower.Status = domain.TowerStatusOffline
				log.Printf(
					"[NetEco] status=offline tower=%s neid=%s reason=site_connect_down connectStatus=%q",
					tower.Name, tower.NetecoNEID, connectRaw,
				)
			case netEcoConnectUp:
				// Só promove para online se não houver degraded activo
				// (alarme/trap crítico em curso).
				if tower.Status == domain.TowerStatusDegraded {
					log.Printf(
						"[NetEco] status=degraded mantido tower=%s neid=%s connectStatus=%q",
						tower.Name, tower.NetecoNEID, connectRaw,
					)
				} else {
					tower.Status = domain.TowerStatusOnline
					log.Printf(
						"[NetEco] status=online tower=%s neid=%s connectStatus=%q",
						tower.Name, tower.NetecoNEID, connectRaw,
					)
				}
			default:
				// siteConnectStatus vazio/desconhecido: comportamento anterior
				// (coleta ok → online), mas regista aviso para afinar o parser.
				if connectRaw != "" {
					log.Printf(
						"[NetEco] siteConnectStatus desconhecido tower=%s value=%q — tratar como up",
						tower.Name, connectRaw,
					)
				}
				if tower.Status == domain.TowerStatusNoData || tower.Status == domain.TowerStatusOffline {
					tower.Status = domain.TowerStatusOnline
				}
				// degraded mantém-se
			}
		}

		tower.UpdatedAt = collectedAt
		if err := s.towerRepo.Upsert(ctx, tower); err != nil {
			log.Printf("[NetEco] erro ao gravar torre %s: %v", tower.Name, err)
		}
		s.recordMetricSnapshot(ctx, tower)
	}
}

func (s *NetEcoIngestService) recordMetricSnapshot(ctx context.Context, tower *domain.Tower) {
	values := map[string]float64{}
	if tower.DCOutputVoltage != nil {
		values["dc_output_voltage"] = *tower.DCOutputVoltage
	}
	if tower.DCLoadCurrent != nil {
		values["dc_load_current"] = *tower.DCLoadCurrent
	}
	if tower.RectifierCurrent != nil {
		values["rectifier_current"] = *tower.RectifierCurrent
	}
	if tower.BatterySOC != nil {
		values["battery_soc"] = *tower.BatterySOC
	}
	if tower.BatterySOH != nil {
		values["battery_soh"] = *tower.BatterySOH
	}
	if tower.BatteryBackupTimeH != nil {
		values["battery_backup_time_h"] = *tower.BatteryBackupTimeH
	}

	if len(values) == 0 {
		return
	}

	metric := &domain.Metric{
		TowerID:     tower.ID,
		CollectedAt: time.Now(),
		Values:      values,
	}
	if err := s.metricSvc.Create(ctx, metric); err != nil {
		log.Printf("[NetEco] erro ao gravar histórico de métricas %s: %v", tower.Name, err)
	}
}

func (s *NetEcoIngestService) applyBatteryStatus(tower *domain.Tower, status *interfaces.BatteryStatus) {
	now := time.Now()

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
	if status.BackupTimeH > 0 {
		bt := status.BackupTimeH
		tower.BatteryBackupTimeH = &bt
	} else {
		tower.BatteryBackupTimeH = nil
	}
	tower.BatteryUpdatedAt = &now
}

func (s *NetEcoIngestService) applyEnergyStatus(tower *domain.Tower, info *interfaces.SiteCounterInfo) {
	tower.DCOutputVoltage = info.DCOutputVoltage
	tower.DCLoadCurrent = info.DCLoadCurrent
	tower.RectifierCurrent = info.RectifierCurrent
}