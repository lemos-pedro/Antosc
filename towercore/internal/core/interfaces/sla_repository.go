package interfaces

import (
	"context"

	"towercore/internal/core/domain"
)

type SLARepository interface {
	GetGlobal(ctx context.Context) (*domain.SLA, error)
}