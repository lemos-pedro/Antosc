package domain

import "time"

// EventType classifica o tipo de evento.
type EventType string

const (
	EventTypeFailure EventType = "failure"
	EventTypeAlarm   EventType = "alarm"
	EventTypeInfo    EventType = "info"
)

// EventSeverity classifica a gravidade do evento.
type EventSeverity string

const (
	EventSeverityInfo     EventSeverity = "info"
	EventSeverityWarning  EventSeverity = "warning"
	EventSeverityCritical EventSeverity = "critical"
)

// EventStatus indica se a condição de alarme que originou o evento ainda
// está ativa ("open") ou já deixou de se verificar ("resolved"). Usado
// para deduplicar alarmes persistentes que, sem isto, criavam um novo
// evento a cada ciclo de poll SNMP (~45s).
type EventStatus string

const (
	EventStatusOpen     EventStatus = "open"
	EventStatusResolved EventStatus = "resolved"
)

// Event representa um alarme ou falha associada a uma torre.
type Event struct {
	ID         string        `json:"event_id"`
	TowerID    string        `json:"tower_id"`
	Type       EventType     `json:"type"`
	Severity   EventSeverity `json:"severity"`
	Message    string        `json:"message"`
	DataSource string        `json:"data_source"`

	OccurredAt time.Time   `json:"occurred_at"`
	CreatedAt  time.Time   `json:"created_at"`
	Status     EventStatus `json:"status"`
	AlarmKey   string      `json:"alarm_key,omitempty"`
	ResolvedAt *time.Time  `json:"resolved_at,omitempty"`
	LastSeenAt time.Time   `json:"last_seen_at"`
}

// IsCritical indica se o evento requer atenção imediata.
func (e *Event) IsCritical() bool {
	return e.Severity == EventSeverityCritical
}

// IsOpen indica se a condição de alarme ainda está ativa.
func (e *Event) IsOpen() bool {
	return e.Status == EventStatusOpen
}
