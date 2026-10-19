package usecase

import (
	"context"

	"github.com/Reddetk/Adev/internal/domain"
)

// ObjectStore is the persistence port of the target system registry.
type ObjectStore interface {
	Save(ctx context.Context, object domain.Object) error
	Get(ctx context.Context, objectID string) (domain.Object, error)
	FindByTargetSystem(ctx context.Context, targetSystemID int64) ([]domain.Object, error)
	Allocate(ctx context.Context, kind domain.Kind) (int, error)
}
