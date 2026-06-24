package interfaces

import (
	"context"
	"errors"
	"time"

	"towercore/internal/core/domain"
)

var ErrTicketNotFound = errors.New("ticket not found")

// TicketFilter define os filtros aceites na listagem de tickets.
type TicketFilter struct {
	Status  string
	TowerID string
	Limit   int
	Offset  int
}

// TicketRepository define o contrato de persistência para tickets.
type TicketRepository interface {
	List(ctx context.Context, filter TicketFilter) ([]domain.Ticket, int, error)
	GetByID(ctx context.Context, id string) (*domain.Ticket, error)
	UpdateStatus(
		ctx context.Context,
		id string,
		status domain.TicketStatus,
		acknowledgedAt, closedAt *time.Time,
		updatedAt time.Time,
	) (*domain.Ticket, error)
}