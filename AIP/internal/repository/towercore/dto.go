package towercore

import "time"

// TowerDTO espelha o payload de GET /api/v1/towers (ver api.md).
type TowerDTO struct {
	TowerID    string    `json:"tower_id"`
	Name       string    `json:"name"`
	Status     string    `json:"status"`
	OperatorID string    `json:"operator_id"`
	RegionID   string    `json:"region_id"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// EventDTO espelha o payload de GET /api/v1/events (endpoint agregado,
// usado por GetEvents). Cada evento inclui o próprio tower_id, já que a
// lista cruza todas as torres -- ao contrário do endpoint por torre
// (GET /towers/{tower_id}/events), onde o ID vem implícito na URL e não
// no corpo da resposta.
type EventDTO struct {
	EventID    string    `json:"event_id"`
	TowerID    string    `json:"tower_id"`
	Type       string    `json:"type"`
	Severity   string    `json:"severity"`
	Message    string    `json:"message"`
	OccurredAt time.Time `json:"occurred_at"`
}

// MetricDTO espelha exatamente towercore/internal/core/domain.Metric,
// devolvido por GET /api/v1/metrics. Note que este endpoint NÃO usa o
// envelope {"data":..., "meta":...} do resto da API — devolve o array
// diretamente, com o total no header X-Total-Count.
//
// Uma métrica é um snapshot por torre com várias grandezas dentro de
// Values (ex.: {"voltage": 53.2, "temperature": 28.1}), não uma
// métrica isolada. Não existe campo de unidade no payload — Feature.Unit
// fica vazio até o towercore expor isso.
type MetricDTO struct {
	ID          string             `json:"metric_id"`
	TowerID     string             `json:"tower_id"`
	CollectedAt time.Time          `json:"collected_at"`
	Values      map[string]float64 `json:"metrics"`
	CreatedAt   time.Time          `json:"created_at"`
}
