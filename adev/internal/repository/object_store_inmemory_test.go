package repository

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/Reddetk/Adev/internal/domain"
)

// inMemoryObjectStore is an in-process registry store for tests.
type inMemoryObjectStore struct {
	mu      sync.Mutex
	objects map[string]domain.Object
	seq     map[domain.Kind]int
}

// newInMemoryObjectStore returns an empty store.
func newInMemoryObjectStore() *inMemoryObjectStore {
	return &inMemoryObjectStore{
		objects: map[string]domain.Object{},
		seq:     map[domain.Kind]int{},
	}
}

// Save writes an object.
func (m *inMemoryObjectStore) Save(ctx context.Context, object domain.Object) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objects[object.ObjectID] = object
	return nil
}

// Get reads an object by id.
func (m *inMemoryObjectStore) Get(ctx context.Context, objectID string) (domain.Object, error) {
	if err := ctx.Err(); err != nil {
		return domain.Object{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	object, ok := m.objects[objectID]
	if !ok {
		return domain.Object{}, fmt.Errorf("%w: %s", domain.ErrObjectNotFound, objectID)
	}
	return object, nil
}

// FindByTargetSystem reads the objects of one target system, ordered by id.
func (m *inMemoryObjectStore) FindByTargetSystem(ctx context.Context, targetSystemID int64) ([]domain.Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []domain.Object
	for _, object := range m.objects {
		if object.TargetSystemID == targetSystemID {
			result = append(result, object)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ObjectID < result[j].ObjectID })
	return result, nil
}

// Allocate returns the next sequence number for a kind.
func (m *inMemoryObjectStore) Allocate(ctx context.Context, kind domain.Kind) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seq[kind]++
	return m.seq[kind], nil
}
