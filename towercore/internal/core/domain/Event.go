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

// Event representa um alarme ou falha associada a uma torre.
type Event struct {
	ID         string        `json:"event_id"`
	TowerID    string        `json:"tower_id"`
	Type       EventType     `json:"type"`
	Severity   EventSeverity `json:"severity"`
	Message    string        `json:"message"`
	OccurredAt time.Time     `json:"occurred_at"`
	CreatedAt  time.Time     `json:"created_at"`
}

// IsCritical indica se o evento requer atenção imediata.
func (e *Event) IsCritical() bool {
	return e.Severity == EventSeverityCritical
}
