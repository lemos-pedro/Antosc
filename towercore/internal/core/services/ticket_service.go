package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
)

type TicketService struct {
	repo      interfaces.TicketRepository
	auditRepo interfaces.AuditRepository
}

func NewTicketService(repo interfaces.TicketRepository, auditRepo ...interfaces.AuditRepository) *TicketService {
	var ar interfaces.AuditRepository
	if len(auditRepo) > 0 {
		ar = auditRepo[0]
	}
	return &TicketService{repo: repo, auditRepo: ar}
}

func (s *TicketService) List(ctx context.Context, filter interfaces.TicketFilter) ([]domain.Ticket, int, error) {
	if filter.Limit <= 0 || filter.Limit > 200 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	return s.repo.List(ctx, filter)
}

func (s *TicketService) Acknowledge(ctx context.Context, actor, ticketID string) (*domain.Ticket, error) {
	ticketID = strings.TrimSpace(ticketID)
	if ticketID == "" {
		return nil, errors.New("ticket_id is required")
	}

	current, err := s.repo.GetByID(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	if current.Status == domain.TicketStatusClosed {
		return nil, errors.New("cannot acknowledge a closed ticket")
	}

	now := time.Now().UTC()
	ticket, err := s.repo.UpdateStatus(ctx, ticketID, domain.TicketStatusAcknowledged, &now, nil, now)
	if err != nil {
		return nil, err
	}
	s.audit(ctx, actor, "ticket.ack", ticket.ID)
	return ticket, nil
}

func (s *TicketService) Close(ctx context.Context, actor, ticketID string) (*domain.Ticket, error) {
	ticketID = strings.TrimSpace(ticketID)
	if ticketID == "" {
		return nil, errors.New("ticket_id is required")
	}

	current, err := s.repo.GetByID(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	if current.Status == domain.TicketStatusClosed {
		// idempotente: fechar um ticket já fechado não é erro.
		return current, nil
	}

	now := time.Now().UTC()
	ticket, err := s.repo.UpdateStatus(ctx, ticketID, domain.TicketStatusClosed, nil, &now, now)
	if err != nil {
		return nil, err
	}
	s.audit(ctx, actor, "ticket.close", ticket.ID)
	return ticket, nil
}

func (s *TicketService) audit(ctx context.Context, actor, action, resourceID string) {
	if s.auditRepo == nil {
		return
	}
	id, err := newUUIDv4()
	if err != nil {
		return
	}
	entry := &domain.AuditLog{
		ID:         id,
		Actor:      safeActor(actor),
		Action:     action,
		Resource:   "ticket",
		ResourceID: resourceID,
		Details:    action + " via api",
		CreatedAt:  time.Now().UTC(),
	}
	_ = s.auditRepo.Create(ctx, entry)
}