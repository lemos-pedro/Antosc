package domain

import "time"

// TicketStatus representa o estado de um ticket de suporte.
type TicketStatus string

const (
	TicketStatusOpen         TicketStatus = "open"
	TicketStatusAcknowledged TicketStatus = "acknowledged"
	TicketStatusClosed       TicketStatus = "closed"
)

// Ticket representa um incidente aberto a partir de um evento crítico.
type Ticket struct {
	ID             string       `json:"ticket_id"`
	TowerID        string       `json:"tower_id"`
	TowerName      string       `json:"tower_name"`
	EventID        string       `json:"event_id"`
	Status         TicketStatus `json:"status"`
	AcknowledgedAt *time.Time   `json:"acknowledged_at,omitempty"`
	ClosedAt       *time.Time   `json:"closed_at,omitempty"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}
