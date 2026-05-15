package interfaces

import (
	"context"
	"time"

	"towercore/internal/core/domain"
)

type AuditFilter struct {
	Actor    string
	Action   string
	Resource string
	From     *time.Time
	To       *time.Time
	Limit    int
	Offset   int
}

type AuditRepository interface {
	Create(ctx context.Context, entry *domain.AuditLog) error
	List(ctx context.Context, filter AuditFilter) ([]domain.AuditLog, int, error)
	GetByID(ctx context.Context, id string) (*domain.AuditLog, error)
}
