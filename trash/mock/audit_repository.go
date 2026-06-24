package mock

import (
	"context"
	"database/sql"
	"slices"
	"sync"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
)

type AuditRepository struct {
	mu   sync.Mutex
	logs []domain.AuditLog
}

func NewAuditRepository() *AuditRepository {
	return &AuditRepository{}
}

func (r *AuditRepository) Create(_ context.Context, entry *domain.AuditLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logs = append(r.logs, *entry)
	return nil
}

func (r *AuditRepository) List(_ context.Context, filter interfaces.AuditFilter) ([]domain.AuditLog, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	filtered := make([]domain.AuditLog, 0, len(r.logs))
	for _, l := range r.logs {
		if filter.Actor != "" && l.Actor != filter.Actor {
			continue
		}
		if filter.Action != "" && l.Action != filter.Action {
			continue
		}
		if filter.Resource != "" && l.Resource != filter.Resource {
			continue
		}
		if filter.From != nil && l.CreatedAt.Before(*filter.From) {
			continue
		}
		if filter.To != nil && l.CreatedAt.After(*filter.To) {
			continue
		}
		filtered = append(filtered, l)
	}

	total := len(filtered)
	start := min(filter.Offset, total)
	end := min(start+filter.Limit, total)
	return slices.Clone(filtered[start:end]), total, nil
}

func (r *AuditRepository) GetByID(_ context.Context, id string) (*domain.AuditLog, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, l := range r.logs {
		if l.ID == id {
			c := l
			return &c, nil
		}
	}
	return nil, sql.ErrNoRows
}
