//go:build tdd

package acceptance

// pr9_registry tests docs/ca-refactor-plan.md section "Registry wiring": an
// accepted HLD proposes its containers, an accepted LLD proposes the layers,
// packages and modules of every container into the target system registry.
// The rerun of the same document must not duplicate an object.

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"testing"

	"github.com/Reddetk/Adev/internal/domain"
	"github.com/Reddetk/Adev/internal/usecase"
)

// registryStore is the in-process registry store of these runs.
type registryStore struct {
	objects map[string]domain.Object
	seq     map[domain.Kind]int
}

func newRegistryStore() *registryStore {
	return &registryStore{objects: map[string]domain.Object{}, seq: map[domain.Kind]int{}}
}

func (f *registryStore) Save(_ context.Context, object domain.Object) error {
	f.objects[object.ObjectID] = object
	return nil
}

func (f *registryStore) Get(_ context.Context, objectID string) (domain.Object, error) {
	object, ok := f.objects[objectID]
	if !ok {
		return domain.Object{}, fmt.Errorf("registry store: no object %s", objectID)
	}
	return object, nil
}

func (f *registryStore) FindByTargetSystem(_ context.Context, targetSystemID int64) ([]domain.Object, error) {
	var result []domain.Object
	for _, object := range f.objects {
		if object.TargetSystemID == targetSystemID {
			result = append(result, object)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ObjectID < result[j].ObjectID })
	return result, nil
}

func (f *registryStore) Allocate(_ context.Context, kind domain.Kind) (int, error) {
	f.seq[kind]++
	return f.seq[kind], nil
}

func (f *registryStore) list(t *testing.T) []domain.Object {
	t.Helper()
	objects, err := f.FindByTargetSystem(context.Background(), testTarget)
	if err != nil {
		t.Fatalf("list registry: %v", err)
	}
	return objects
}

func countKind(objects []domain.Object, kind domain.Kind) int {
	n := 0
	for _, object := range objects {
		if object.Kind == kind {
			n++
		}
	}
	return n
}

func byKind(t *testing.T, objects []domain.Object, kind domain.Kind) domain.Object {
	t.Helper()
	for _, object := range objects {
		if object.Kind == kind {
			return object
		}
	}
	t.Fatalf("no %s object among %+v", kind, objects)
	return domain.Object{}
}

// TestStageRunProposesTheHLDContainers: stage 03 stores the HLD and the
// registry holds one container object per containers entry.
func TestStageRunProposesTheHLDContainers(t *testing.T) {
	fixture := newFixture(t, respondWith(t, map[domain.FormType]json.RawMessage{
		domain.TypeHLD: validHLD(t),
	}))
	store := newRegistryStore()
	fixture.stages = fixture.stages.WithRegistry(usecase.NewRegistryService(store))

	fixture.submit(t, "FRM-1-02", domain.TypeSRS, "02", validSRS(t))
	fixture.run(t, "03")

	objects := store.list(t)
	if n := countKind(objects, domain.KindContainer); n != 1 {
		t.Fatalf("want 1 container object, got %d: %+v", n, objects)
	}
	container := byKind(t, objects, domain.KindContainer)
	if container.Name != "obj-container-1" {
		t.Fatalf("want the model objectId as the object name, got %q", container.Name)
	}
	if container.Fields["tech"] != "go" || container.Fields["boundedContextId"] != "bc-1" {
		t.Fatalf("bad container fields: %+v", container.Fields)
	}
	if container.IntroducedBy.FormType != domain.TypeHLD {
		t.Fatalf("the container must name the HLD that introduced it: %+v", container.IntroducedBy)
	}
}

// TestStageRunProposesTheLLDLayoutAndStaysIdempotent: stage 05 stores the LLD
// and the registry holds the layers, packages and modules; a second registration
// pass of the same document changes nothing.
func TestStageRunProposesTheLLDLayoutAndStaysIdempotent(t *testing.T) {
	fixture := newFixture(t, respondWith(t, map[domain.FormType]json.RawMessage{
		domain.TypeHLD: validHLD(t),
		domain.TypeLLD: validLLD(t),
	}))
	store := newRegistryStore()
	fixture.stages = fixture.stages.WithRegistry(usecase.NewRegistryService(store))

	fixture.submit(t, "FRM-1-03", domain.TypeHLD, "03", validHLD(t))
	fixture.submit(t, "FRM-1-04-ADR", domain.TypeADR, "04", validADR(t))
	fixture.run(t, "05")

	objects := store.list(t)
	if n := countKind(objects, domain.KindLayer); n != 1 {
		t.Fatalf("want 1 layer object, got %d: %+v", n, objects)
	}
	if layer := byKind(t, objects, domain.KindLayer); layer.Name != "app" || layer.ContainerID != "obj-container-1" {
		t.Fatalf("bad layer object: %+v", layer)
	}
	if n := countKind(objects, domain.KindPackage); n != 1 {
		t.Fatalf("want 1 package object, got %d: %+v", n, objects)
	}
	if pkg := byKind(t, objects, domain.KindPackage); pkg.Path != "internal/app" || pkg.ContainerID != "obj-container-1" {
		t.Fatalf("bad package object: %+v", pkg)
	}
	if module := byKind(t, objects, domain.KindModule); module.Name != "obj-module-1" || module.Path != "internal/app/book" {
		t.Fatalf("bad module object: %+v", module)
	}

	before := len(store.objects)
	fixture.run(t, "05")
	if after := len(store.objects); after != before {
		t.Fatalf("the rerun must not duplicate objects: %d -> %d", before, after)
	}
}
