package usecase

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/Reddetk/Adev/internal/domain"
)

// fakeObjectStore is an in-process double of ObjectStore.
type fakeObjectStore struct {
	mu      sync.Mutex
	objects map[string]domain.Object
	seq     map[domain.Kind]int
}

func newFakeObjectStore() *fakeObjectStore {
	return &fakeObjectStore{
		objects: map[string]domain.Object{},
		seq:     map[domain.Kind]int{},
	}
}

func (f *fakeObjectStore) Save(_ context.Context, object domain.Object) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.objects[object.ObjectID] = object
	return nil
}

func (f *fakeObjectStore) Get(_ context.Context, objectID string) (domain.Object, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	object, ok := f.objects[objectID]
	if !ok {
		return domain.Object{}, fmt.Errorf("%w: %s", domain.ErrObjectNotFound, objectID)
	}
	return object, nil
}

func (f *fakeObjectStore) FindByTargetSystem(_ context.Context, targetSystemID int64) ([]domain.Object, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var result []domain.Object
	for _, object := range f.objects {
		if object.TargetSystemID == targetSystemID {
			result = append(result, object)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ObjectID < result[j].ObjectID })
	return result, nil
}

func (f *fakeObjectStore) Allocate(_ context.Context, kind domain.Kind) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq[kind]++
	return f.seq[kind], nil
}
