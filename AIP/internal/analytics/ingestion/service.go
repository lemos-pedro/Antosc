package ingestion

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/antosc/aip/internal/repository/postgres"
	"github.com/antosc/aip/internal/repository/towercore"
)

// initialLookback e usado so no primeiro ciclo (antes de termos qualquer
// "ultima execucao" conhecida), para nao pedir o historico completo ao
// towercore logo no arranque do processo.
const initialLookback = 24 * time.Hour

type ingestionService struct {
	towercore towercore.Client
	features  postgres.FeatureRepository
	events    postgres.EventRepository
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
	features postgres.FeatureRepository,
	events postgres.EventRepository,
	log *slog.Logger,
) Service {
	return &ingestionService{
		towercore:        client,
		features:         features,
		events:           events,
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

	if err := s.watchTowerStatus(ctx, towers); err != nil {
		// Não aborta o ciclo por causa disto -- métricas/eventos
		// normais continuam a ser mais importantes que o watchdog de
		// status. Só regista o problema.
		s.log.Warn("watchdog de status das torres falhou", "err", err)
	}

	metrics, err := s.towercore.GetMetrics(ctx, s.lastMetricsSince)
	if err != nil {
		return err
	}
	if err := s.saveMetrics(ctx, metrics); err != nil {
		return err
	}
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

	s.log.Info("ciclo de ingestao concluido", "towers", len(towers), "metrics", len(metrics), "events", len(events))
	return nil
}

// degradedStatuses são os status de towercore que o AIP trata como
// "condição a vigiar" -- cria/toca um evento aberto em ai_events.
// "online" é o único status considerado saudável; qualquer coisa fora
// disto (degraded, offline, ou um valor novo que apareça no futuro)
// entra aqui, para não passar nada em claro por engano.
func isUnhealthyStatus(status string) bool {
	return status != "online" && status != ""
}

// alarmKeyForStatus identifica a condição de forma estável por status,
// para o dedup (tower_id, alarm_key) do CreateOrTouch funcionar mesmo
// que o status mude entre duas variantes "más" (ex. degraded -> offline
// sem passar por online) -- nesse caso, o alarm_key muda, o antigo fica
// por resolver deliberadamente (a torre continua com problema, só que
// pior) e o novo é criado. Só volta tudo a "resolved" quando o status
// for mesmo "online".
func alarmKeyForStatus(status string) string {
	return "tower_status:" + status
}

// watchTowerStatus é o watchdog que falta no towercore (Prioridade #1
// do projeto: torres "degradada"/"offline" sem alarme real por trás).
// Não substitui a correção no towercore -- é um paliativo que o AIP
// consegue fazer sozinho, sem depender do código do towercore, a partir
// do "status" que GetTowers já devolve corretamente.
func (s *ingestionService) watchTowerStatus(ctx context.Context, towers []towercore.TowerDTO) error {
	for _, t := range towers {
		if t.TowerID == "" {
			continue
		}

		if isUnhealthyStatus(t.Status) {
			severity := "warning"
			if t.Status == "offline" {
				severity = "critical"
			}

			err := s.events.CreateOrTouch(ctx, postgres.OpenEvent{
				TowerID:  t.TowerID,
				AlarmKey: alarmKeyForStatus(t.Status),
				Type:     "tower_status",
				Severity: severity,
				Message:  fmt.Sprintf("torre com status %q reportado pelo towercore", t.Status),
			})
			if err != nil {
				s.log.Warn("falha ao criar/tocar evento de status", "tower_id", t.TowerID, "status", t.Status, "err", err)
			}
			continue
		}

		// status == "online": resolve qualquer evento aberto de status
		// mau que ainda exista para esta torre. Tenta os dois valores
		// conhecidos -- Resolve é no-op se não houver nada aberto,
		// nunca falha por "não havia nada para resolver".
		for _, badStatus := range []string{"degraded", "offline"} {
			if err := s.events.Resolve(ctx, t.TowerID, alarmKeyForStatus(badStatus)); err != nil {
				s.log.Warn("falha ao resolver evento de status", "tower_id", t.TowerID, "err", err)
			}
		}
	}

	return nil
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

// saveEvents grava os eventos que o towercore reporta como abertos.
// GET /api/v1/events devolve um snapshot do que está aberto AGORA, não
// um delta desde o último ciclo -- por isso um evento que continua
// aberto aparece de novo em todos os ciclos seguintes. Um INSERT cego
// (a versão anterior) duplicava a mesma linha a cada 5 minutos. Usa-se
// CreateOrTouch com o event_id do towercore como alarm_key: mesma
// lógica de dedup do watchdog de status (watchTowerStatus), só que a
// chave vem de fora em vez de ser construída aqui.
func (s *ingestionService) saveEvents(ctx context.Context, events []towercore.EventDTO) error {
	if len(events) == 0 {
		return nil
	}

	skipped := 0
	for _, e := range events {
		if e.TowerID == "" || e.EventID == "" {
			skipped++
			continue
		}

		err := s.events.CreateOrTouch(ctx, postgres.OpenEvent{
			TowerID:  e.TowerID,
			AlarmKey: "towercore_event:" + e.EventID,
			Type:     e.Type,
			Severity: e.Severity,
			Message:  e.Message,
		})
		if err != nil {
			s.log.Warn("falha ao gravar evento do towercore", "event_id", e.EventID, "tower_id", e.TowerID, "err", err)
		}
	}

	if skipped > 0 {
		s.log.Warn("eventos sem tower_id/event_id ignorados", "count", skipped)
	}

	return nil
}
