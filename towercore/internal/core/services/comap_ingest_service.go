package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"towercore/internal/adapters/comap"
	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
)

// ComapIngestService persiste a última leitura de telemetria ComAp (Modbus)
// conhecida por torre e mantém o status da torre coerente quando o
// controlador do gerador fica inacessível. Segue o mesmo padrão de
// SNMPIngestService/NagiosIngestService: nunca deixa o status "preso" no
// último valor bom quando a coleta falha (ver MarkUnreachable).
//
// NOTA: esta é a implementação que faltava. O ficheiro anteriormente
// existente aqui tinha `package service` (typo) e continha uma cópia
// duplicada do ComapScheduler colada por engano — isso impedia a
// compilação de todo o módulo. Foi corrigido em 2026-07-28.
type ComapIngestService struct {
	readingRepo  interfaces.ComapReadingRepository
	eventService *EventService
	ticketSvc    *TicketService
	towerUpdater TowerStatusUpdater
}

func NewComapIngestService(
	readingRepo interfaces.ComapReadingRepository,
	eventService *EventService,
	ticketSvc *TicketService,
	towerUpdater TowerStatusUpdater,
) *ComapIngestService {
	return &ComapIngestService{
		readingRepo:  readingRepo,
		eventService: eventService,
		ticketSvc:    ticketSvc,
		towerUpdater: towerUpdater,
	}
}

// Ingest lê a telemetria atual via comap.Reader e grava-a (upsert) como a
// última leitura conhecida da torre. Campos que falharam a leitura ficam
// nil — nunca são fabricados como zero (princípio "no fabricated data",
// já seguido pelo Reader e por interfaces.ComapReading).
//
// Coleta com sucesso implica que o controlador está a comunicar: o status
// da torre é reposto para online aqui, ANTES de qualquer outra escrita,
// seguindo a mesma ordem usada no SNMPIngestService (status primeiro,
// para nunca ficar desatualizado se algo falhar depois).
func (s *ComapIngestService) Ingest(ctx context.Context, towerID string, reader *comap.Reader) error {
	if strings.TrimSpace(towerID) == "" {
		return errors.New("tower_id is required")
	}
	if reader == nil {
		return errors.New("reader is required")
	}

	metrics, err := reader.Read(ctx)
	if err != nil {
		return err
	}

	collectedAt := metrics.CollectedAt
	if collectedAt.IsZero() {
		collectedAt = time.Now().UTC()
	}

	if s.towerUpdater != nil {
		if err := s.towerUpdater.UpdateStatus(ctx, towerID, domain.TowerStatusOnline); err != nil {
			return err
		}
	}

	reading := &interfaces.ComapReading{
		TowerID:         towerID,
		FuelLiters:      metrics.FuelLevelPct,
		FuelPercent:     metrics.FuelLevelPct,
		BatteryVoltageV: metrics.BatteryVoltageV,
		RunHoursTotal:   metrics.RunHoursTotal,
		CollectedAt:     &collectedAt,
	}

	return s.readingRepo.Upsert(ctx, reading)
}

// MarkUnreachable marca a torre como offline quando a coleta ComAp falha
// (timeout de conexão Modbus, controlador desligado, endpoint sem
// resposta, etc). Chamado pelo ComapScheduler em qualquer ponto de falha
// do ciclo de coleta — é isto que impede o "estado preso" quando perdemos
// comunicação com o gerador.
func (s *ComapIngestService) MarkUnreachable(ctx context.Context, towerID string) error {
	if strings.TrimSpace(towerID) == "" {
		return errors.New("tower_id is required")
	}
	if s.towerUpdater == nil {
		return nil
	}
	return s.towerUpdater.UpdateStatus(ctx, towerID, domain.TowerStatusOffline)
}