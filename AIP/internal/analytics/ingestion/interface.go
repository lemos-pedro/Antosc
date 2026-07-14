package ingestion

import "context"

type Service interface {
	Collect(ctx context.Context) error
}
