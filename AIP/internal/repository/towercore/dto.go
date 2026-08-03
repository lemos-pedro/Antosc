package towercore

import "time"

// TowerDTO espelha o payload de GET /api/v1/towers (ver api.md).
// Nota: o payload real tem mais campos (operators, snmp_*, neteco_*) que
// não são usados pelo AIP e ficam de fora deste DTO de propósito.
type TowerDTO struct {
	TowerID          string    `json:"tower_id"`
	Name             string    `json:"name"`
	Status           string    `json:"status"`
	OperatorID       string    `json:"operator_id"`
	RegionID         string    `json:"region_id"`
	Vendor           string    `json:"vendor"` // eltek | huawei | enetek -- usado para escolher o adaptador no /predict
	Availability7d   float64   `json:"availability_7d"`
	Availability30d  float64   `json:"availability_30d"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// EventDTO espelha o payload de GET /api/v1/towers/{tower_id}/events.
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
