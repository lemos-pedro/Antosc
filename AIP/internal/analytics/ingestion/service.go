package ingestion

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/antosc/aip/internal/alerts"
	"github.com/antosc/aip/internal/repository/generator"
	"github.com/antosc/aip/internal/repository/postgres"
	"github.com/antosc/aip/internal/repository/towercore"
)

// initialLookback e usado so no primeiro ciclo (antes de termos qualquer
// "ultima execucao" conhecida), para nao pedir o historico completo ao
// towercore logo no arranque do processo.
const initialLookback = 24 * time.Hour

type ingestionService struct {
	towercore towercore.Client
	generator generator.Client // opcional; nil desliga a leitura de combustível/gerador
	features  postgres.FeatureRepository
	events    postgres.EventRepository
	incidents postgres.IncidentCauseRepository
	towers    postgres.TowerRepository
	alerts    *alerts.Engine // opcional; nil desliga a avaliação de alertas de limiar
	log       *slog.Logger

	// lastMetricsSince acompanha ate onde ja foi lido GET /api/v1/metrics,
	// para cada ciclo so pedir o que e novo desde o ciclo anterior, em vez
	// do historico completo a cada 5 minutos. Como Collect() e sempre
	// chamado sequencialmente pelo scheduler (um ticker, uma goroutine),
	// nao precisa de lock.
	lastMetricsSince time.Time
}

func NewService(
	client towercore.Client,
	generatorClient generator.Client,
	features postgres.FeatureRepository,
	events postgres.EventRepository,
	incidents postgres.IncidentCauseRepository,
	towers postgres.TowerRepository,
	alertEngine *alerts.Engine,
	log *slog.Logger,
) Service {
	return &ingestionService{
		towercore:        client,
		generator:        generatorClient,
		features:         features,
		events:           events,
		incidents:        incidents,
		towers:           towers,
		alerts:           alertEngine,
		log:              log,
		lastMetricsSince: time.Now().Add(-initialLookback),
	}
}

// Collect corre um ciclo de ingestao: le o estado atual do towercore
// (torres, metricas desde o ultimo ciclo, eventos) e grava
// features/eventos IA.
func (s *ingestionService) Collect(ctx context.Context) error {
	cycleStart := time.Now()

	towers, err := s.towercore.GetTowers(ctx)
	if err != nil {
		return err
	}
	s.log.Info("towers coletadas", "count", len(towers))
	s.saveTowers(ctx, towers)

	metrics, err := s.towercore.GetMetrics(ctx, s.lastMetricsSince)
	if err != nil {
		return err
	}
	if err := s.saveMetrics(ctx, metrics); err != nil {
		return err
	}

	generatorReadings := s.collectGeneratorReadings(ctx, towers)
	s.evaluateAlerts(ctx, metrics, generatorReadings)
	// So avanca o marcador depois de gravar com sucesso -- se este ciclo
	// falhar a meio, o proximo tenta outra vez a partir do mesmo ponto.
	s.lastMetricsSince = cycleStart

	events, err := s.towercore.GetEvents(ctx)
	if err != nil {
		return err
	}
	if err := s.saveEvents(ctx, events); err != nil {
		return err
	}
	s.detectIncidents(ctx, events)

	s.log.Info("ciclo de ingestao concluido", "towers", len(towers), "metrics", len(metrics), "events", len(events))
	return nil
}

// saveTowers atualiza a cache local (tabela towers) com o que o towercore
// devolveu -- sobretudo para termos o vendor disponível localmente sem
// termos de chamar o towercore outra vez sempre que o serviço de previsão
// precisar de saber que adaptador usar. Falha aqui não interrompe o ciclo
// -- é uma cache, não a fonte de verdade.
func (s *ingestionService) saveTowers(ctx context.Context, towers []towercore.TowerDTO) {
	if s.towers == nil || len(towers) == 0 {
		return
	}

	batch := make([]postgres.Tower, 0, len(towers))
	for _, t := range towers {
		batch = append(batch, postgres.Tower{
			TowerID:         t.TowerID,
			Name:            t.Name,
			Vendor:          t.Vendor,
			OperatorID:      t.OperatorID,
			RegionID:        t.RegionID,
			Availability7d:  sql.NullFloat64{Float64: t.Availability7d, Valid: true},
			Availability30d: sql.NullFloat64{Float64: t.Availability30d, Valid: true},
		})
	}

	if err := s.towers.UpsertBatch(ctx, batch); err != nil {
		s.log.Warn("falha ao atualizar cache local de towers", "err", err)
	}
}

// saveMetrics achata cada snapshot (uma torre, N grandezas em Values) em
// N linhas de ai_features -- uma por grandeza. Nao ha campo de unidade no
// payload do towercore, por isso Feature.Unit fica vazio (nao inventamos
// unidades).
func (s *ingestionService) saveMetrics(ctx context.Context, metrics []towercore.MetricDTO) error {
	if len(metrics) == 0 {
		return nil
	}

	batch := make([]postgres.Feature, 0, len(metrics))
	for _, m := range metrics {
		for name, value := range m.Values {
			batch = append(batch, postgres.Feature{
				TowerID:   m.TowerID,
				Name:      name,
				Value:     value,
				Unit:      "",
				CreatedAt: m.CollectedAt,
			})
		}
	}

	return s.features.SaveBatch(ctx, batch)
}

// evaluateAlerts converte as métricas do towercore + as leituras do gerador
// em leituras uniformes e passa-as ao motor de alertas, que despacha para o
// Teams o que ultrapassar limiar. Sem alertEngine configurado, não faz nada.
// battery_remaining_percent é derivado aqui (battery_remaining_ / battery_total_
// * 100) porque o towercore só reporta os valores absolutos, não a percentagem.
func (s *ingestionService) evaluateAlerts(ctx context.Context, metrics []towercore.MetricDTO, generatorReadings []alerts.Reading) {
	if s.alerts == nil {
		return
	}

	readings := make([]alerts.Reading, 0, len(metrics)+len(generatorReadings))
	for _, m := range metrics {
		for name, value := range m.Values {
			readings = append(readings, alerts.Reading{TowerID: m.TowerID, FeatureName: name, Value: value})
		}

		if total, ok := m.Values["battery_total_"]; ok && total > 0 {
			if remaining, ok := m.Values["battery_remaining_"]; ok {
				readings = append(readings, alerts.Reading{
					TowerID:     m.TowerID,
					FeatureName: "battery_remaining_percent",
					Value:       (remaining / total) * 100,
				})
			}
		}
	}
	readings = append(readings, generatorReadings...)

	s.alerts.Evaluate(ctx, readings)
}

// collectGeneratorReadings lê o serviço ComAp por torre. É N+1 chamadas
// (uma por site) tal como GetEvents -- aceitável para já dado o volume;
// candidato a endpoint agregado se crescer. Sites sem gerador reportado
// (generator.ErrNoGenerator) são ignorados sem gerar erro nem log de ruído.
func (s *ingestionService) collectGeneratorReadings(ctx context.Context, towers []towercore.TowerDTO) []alerts.Reading {
	if s.generator == nil {
		return nil
	}

	var readings []alerts.Reading
	for _, t := range towers {
		energy, err := s.generator.GetEnergy(ctx, t.TowerID)
		if err != nil {
			if err != generator.ErrNoGenerator {
				s.log.Warn("falha ao ler energia do gerador", "tower_id", t.TowerID, "err", err)
			}
			continue
		}

		readings = append(readings,
			alerts.Reading{TowerID: t.TowerID, FeatureName: "fuel_liters", Value: energy.FuelLiters},
			alerts.Reading{TowerID: t.TowerID, FeatureName: "fuel_percent", Value: energy.FuelPercent},
		)
	}
	return readings
}

func (s *ingestionService) saveEvents(ctx context.Context, events []towercore.EventDTO) error {
	if len(events) == 0 {
		return nil
	}

	batch := make([]postgres.AIEvent, 0, len(events))
	skipped := 0
	for _, e := range events {
		if e.TowerID == "" {
			skipped++
			continue
		}
		batch = append(batch, postgres.AIEvent{
			TowerID:   e.TowerID,
			Type:      e.Type,
			Severity:  e.Severity,
			Message:   e.Message,
			CreatedAt: e.OccurredAt,
		})
	}
	if skipped > 0 {
		s.log.Warn("eventos ignorados por tower_id vazio", "count", skipped)
	}
	if len(batch) == 0 {
		return nil
	}

	return s.events.SaveBatch(ctx, batch)
}

// isSiteDown identifica eventos de queda de site vindos do towercore.
// Regra simples baseada em tipo/severidade; quando o modelo de deteção de
// anomalias (python/models) estiver ligado, esta função passa a ser
// substituída/complementada pela previsão do ML em vez de só regras fixas.
func isSiteDown(e towercore.EventDTO) bool {
	return e.Type == "site_down" || e.Type == "power_loss" || e.Severity == "critical"
}

// detectIncidents cria um registo em site_incident_causes para cada evento de
// queda, com a causa provável = mensagem do evento do towercore. Fica em
// "pending_confirmation" até o O&M confirmar ou corrigir via
// POST /api/v1/incidents/{id}/confirm. Falhas aqui são só registadas em log
// -- não devem interromper o ciclo de ingestão, que já gravou os dados brutos.
func (s *ingestionService) detectIncidents(ctx context.Context, events []towercore.EventDTO) {
	for _, e := range events {
		if !isSiteDown(e) {
			continue
		}

		_, err := s.incidents.CreateFromPrediction(ctx, postgres.IncidentCause{
			TowerID:           e.TowerID,
			IncidentStartedAt: e.OccurredAt,
			MLPredictedCause: sql.NullString{
				String: e.Message,
				Valid:  e.Message != "",
			},
		})
		if err != nil {
			s.log.Warn("falha ao registar causa de incidente", "tower_id", e.TowerID, "err", err)
		}
	}
}
