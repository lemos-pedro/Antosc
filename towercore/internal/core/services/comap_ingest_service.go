package services

import (
	"context"
	"fmt"

	"towercore/internal/adapters/comap"
	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
)
 
// Threshold de negócio para telemetria ComAp. Combustível é o único campo
// com threshold definido nesta primeira versão — fuel_percent e
// battery_voltage estão marcados StatusUnconfirmed no comap.Profile e não
// geram eventos/tickets até validação em campo, para não repetir o problema
// de falsos "Degradada" já visto no profile Eltek (threshold mal calibrado).
const fuelLowLiters = 50.0 // TODO: ajustar por site após validação de campo (depende do tamanho do tanque)

// ComapModbusReader é o contrato mínimo que o serviço precisa do adapter
// comap (permite mock em testes sem depender do driver Modbus real).
type ComapModbusReader interface {
	Read(ctx context.Context) (*comap.Metrics, error)
}

// ComapIngestService orquestra a leitura de telemetria ComAp para uma torre:
// persiste sempre a última leitura (via ComapReadingRepository, para o
// frontend), e aplica o padrão de dedup de eventos apenas para os campos
// já validados (fuel_liters).
type ComapIngestService struct {
	readings interfaces.ComapReadingRepository
	events   *EventService
	tickets  *TicketService
}

// NewComapIngestService cria o serviço de ingestão ComAp.
func NewComapIngestService(readings interfaces.ComapReadingRepository, events *EventService, tickets *TicketService) *ComapIngestService {
	return &ComapIngestService{readings: readings, events: events, tickets: tickets}
}

// Ingest lê a telemetria de um endpoint ComAp (via reader já configurado
// para a torre/IP correto — ver tower_endpoints), persiste a leitura para
// consumo do frontend, e avalia thresholds. Campos nil (registo inativo,
// não configurado, ou não confirmado em campo) são ignorados na avaliação
// de threshold — nunca tratados como zero — mas ainda assim persistidos
// como nil, para o frontend mostrar "—" corretamente.
func (s *ComapIngestService) Ingest(ctx context.Context, towerID string, reader ComapModbusReader) error {
	metrics, readErr := reader.Read(ctx)
	if metrics == nil {
		return fmt.Errorf("comap ingest: leitura falhou completamente para torre %s: %w", towerID, readErr)
	}

	// Persiste sempre a última leitura conhecida (mesmo que parcial),
	// para o separador Energia no frontend nunca depender do threshold.
	if err := s.readings.Upsert(ctx, &interfaces.ComapReading{
		TowerID:         towerID,
		FuelLiters:      metrics.FuelLiters,
		FuelPercent:     metrics.FuelPercent,
		BatteryVoltageV: metrics.BatteryVoltageV,
		RunHoursTotal:   metrics.RunHoursTotal,
		CollectedAt:     &metrics.CollectedAt,
	}); err != nil {
		return fmt.Errorf("comap ingest: upsert reading tower %s: %w", towerID, err)
	}

	alarmKey := fmt.Sprintf("comap:%s:fuel_low", towerID)

	switch {
	case metrics.FuelLiters != nil && *metrics.FuelLiters < fuelLowLiters:
		event := &domain.Event{
			TowerID:  towerID,
			Type:     domain.EventTypeAlarm,
			Severity: domain.EventSeverityWarning,
			Message:  fmt.Sprintf("Combustível baixo: %.1f L", *metrics.FuelLiters),
		}
		created, isNew, err := s.events.CreateOrTouch(ctx, event, alarmKey)
		if err != nil {
			return fmt.Errorf("comap ingest: CreateOrTouch fuel_low: %w", err)
		}
		if isNew {
			if _, err := s.tickets.Create(ctx, towerID, created.ID); err != nil {
				return fmt.Errorf("comap ingest: TicketService.Create fuel_low: %w", err)
			}
		}

	case metrics.FuelLiters != nil:
		if err := s.events.Resolve(ctx, towerID, alarmKey); err != nil {
			return fmt.Errorf("comap ingest: Resolve fuel_low: %w", err)
		}
	}

	// fuel_percent e battery_voltage: StatusUnconfirmed no Profile.
	// Persistidos acima para o frontend, mas sem avaliação de
	// threshold/evento até confirmação em campo — ver docs/definições.md.

	if readErr != nil {
		return fmt.Errorf("comap ingest: leitura parcial para torre %s: %w", towerID, readErr)
	}
	return nil
}