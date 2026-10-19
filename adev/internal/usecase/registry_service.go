package usecase

import (
	"context"
	"fmt"

	"github.com/Reddetk/Adev/internal/domain"
)

// RegistryService applies the registry rules: id allocation and the object lifecycle.
type RegistryService struct {
	store ObjectStore
}

// NewRegistryService builds a registry service.
func NewRegistryService(store ObjectStore) *RegistryService {
	return &RegistryService{store: store}
}

// Propose registers a new object with status planned and returns it with its id.
func (s *RegistryService) Propose(ctx context.Context, targetSystemID int64, kind domain.Kind, name, path, containerID string, by domain.Ref) (domain.Object, error) {
	if targetSystemID < 1 {
		return domain.Object{}, fmt.Errorf("propose: targetSystemId must be positive")
	}
	if !kind.Valid() {
		return domain.Object{}, fmt.Errorf("propose: unknown kind %q", kind)
	}
	if name == "" {
		return domain.Object{}, fmt.Errorf("propose: name is empty")
	}
	if by.FormID == "" {
		return domain.Object{}, fmt.Errorf("propose: introducedBy.formId is empty")
	}

	seq, err := s.store.Allocate(ctx, kind)
	if err != nil {
		return domain.Object{}, fmt.Errorf("propose: allocate id: %w", err)
	}

	object := domain.Object{
		ObjectID:       fmt.Sprintf("OBJ-%s-%03d", kind, seq),
		TargetSystemID: targetSystemID,
		Kind:           kind,
		Name:           name,
		Path:           path,
		ContainerID:    containerID,
		Status:         domain.ObjectPlanned,
		IntroducedBy:   by,
		ChangedBy:      []domain.Ref{},
	}
	if err := s.store.Save(ctx, object); err != nil {
		return domain.Object{}, fmt.Errorf("propose: save: %w", err)
	}
	return object, nil
}

// Activate moves an object from planned to active.
func (s *RegistryService) Activate(ctx context.Context, objectID string, by domain.Ref) (domain.Object, error) {
	return s.transition(ctx, objectID, domain.ObjectActive, by)
}

// Remove moves an object from planned or active to removed.
func (s *RegistryService) Remove(ctx context.Context, objectID string, by domain.Ref) (domain.Object, error) {
	return s.transition(ctx, objectID, domain.ObjectRemoved, by)
}

// Get reads one object.
func (s *RegistryService) Get(ctx context.Context, objectID string) (domain.Object, error) {
	return s.store.Get(ctx, objectID)
}

// List reads the objects of one target system.
func (s *RegistryService) List(ctx context.Context, targetSystemID int64) ([]domain.Object, error) {
	return s.store.FindByTargetSystem(ctx, targetSystemID)
}

func (s *RegistryService) transition(ctx context.Context, objectID string, to domain.ObjectStatus, by domain.Ref) (domain.Object, error) {
	object, err := s.store.Get(ctx, objectID)
	if err != nil {
		return domain.Object{}, err
	}
	if !domain.CanChange(object.Status, to) {
		return domain.Object{}, fmt.Errorf("%w: %s to %s", domain.ErrInvalidTransition, object.Status, to)
	}
	object.Status = to
	object.ChangedBy = append(object.ChangedBy, by)
	if err := s.store.Save(ctx, object); err != nil {
		return domain.Object{}, fmt.Errorf("transition: save: %w", err)
	}
	return object, nil
}
