package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/Reddetk/Adev/internal/domain"
)

var testRef = domain.Ref{FormID: "FRM-1", FormType: domain.TypeHLD, FormVersion: 1}

func TestRegistryProposeAllocatesPerKind(t *testing.T) {
	service := NewRegistryService(newFakeObjectStore())
	ctx := context.Background()

	first, err := service.Propose(ctx, 1, domain.KindContainer, "user-service", "", "", testRef)
	if err != nil {
		t.Fatalf("propose container: %v", err)
	}
	if first.ObjectID != "OBJ-container-001" || first.Status != domain.ObjectPlanned {
		t.Fatalf("unexpected object: %+v", first)
	}

	module, err := service.Propose(ctx, 1, domain.KindModule, "token-validator", "internal/auth/token_validator.go", first.ObjectID, testRef)
	if err != nil {
		t.Fatalf("propose module: %v", err)
	}
	if module.ObjectID != "OBJ-module-001" {
		t.Fatalf("unexpected id %q", module.ObjectID)
	}

	second, err := service.Propose(ctx, 1, domain.KindContainer, "api-gateway", "", "", testRef)
	if err != nil {
		t.Fatalf("propose second container: %v", err)
	}
	if second.ObjectID != "OBJ-container-002" {
		t.Fatalf("unexpected id %q", second.ObjectID)
	}
}

func TestRegistryLifecycleActiveThenRemoved(t *testing.T) {
	service := NewRegistryService(newFakeObjectStore())
	ctx := context.Background()

	object, _ := service.Propose(ctx, 1, domain.KindModule, "auth", "", "", testRef)
	active, err := service.Activate(ctx, object.ObjectID, domain.Ref{FormID: "FRM-9", FormType: domain.TypeVerdict, FormVersion: 1})
	if err != nil {
		t.Fatalf("activate: %v", err)
	}
	if active.Status != domain.ObjectActive || len(active.ChangedBy) != 1 {
		t.Fatalf("unexpected active object: %+v", active)
	}

	removed, err := service.Remove(ctx, object.ObjectID, testRef)
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if removed.Status != domain.ObjectRemoved || len(removed.ChangedBy) != 2 {
		t.Fatalf("unexpected removed object: %+v", removed)
	}
}

func TestRegistryRejectsInvalidTransitions(t *testing.T) {
	service := NewRegistryService(newFakeObjectStore())
	ctx := context.Background()

	object, _ := service.Propose(ctx, 1, domain.KindModule, "auth", "", "", testRef)
	if _, err := service.Activate(ctx, object.ObjectID, testRef); err != nil {
		t.Fatalf("first activate: %v", err)
	}
	if _, err := service.Activate(ctx, object.ObjectID, testRef); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("want ErrInvalidTransition, got %v", err)
	}
	if _, err := service.Remove(ctx, object.ObjectID, testRef); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := service.Remove(ctx, object.ObjectID, testRef); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("want ErrInvalidTransition, got %v", err)
	}
}

func TestRegistryProposeRejectsBadInput(t *testing.T) {
	service := NewRegistryService(newFakeObjectStore())
	ctx := context.Background()

	if _, err := service.Propose(ctx, 1, domain.Kind("wormhole"), "x", "", "", testRef); err == nil {
		t.Fatal("want an error for an unknown kind")
	}
	if _, err := service.Propose(ctx, 1, domain.KindModule, "", "", "", testRef); err == nil {
		t.Fatal("want an error for an empty name")
	}
	if _, err := service.Propose(ctx, 1, domain.KindModule, "a", "", "", domain.Ref{}); err == nil {
		t.Fatal("want an error for an empty introducedBy")
	}
}

func TestRegistryListsOnlyOneTargetSystem(t *testing.T) {
	service := NewRegistryService(newFakeObjectStore())
	ctx := context.Background()

	if _, err := service.Propose(ctx, 1, domain.KindModule, "a", "", "", testRef); err != nil {
		t.Fatalf("propose: %v", err)
	}
	if _, err := service.Propose(ctx, 2, domain.KindModule, "b", "", "", testRef); err != nil {
		t.Fatalf("propose: %v", err)
	}

	objects, err := service.List(ctx, 1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(objects) != 1 || objects[0].TargetSystemID != 1 {
		t.Fatalf("unexpected list: %+v", objects)
	}
}

func TestRegistryGetUnknownObject(t *testing.T) {
	service := NewRegistryService(newFakeObjectStore())
	if _, err := service.Get(context.Background(), "OBJ-module-404"); !errors.Is(err, domain.ErrObjectNotFound) {
		t.Fatalf("want ErrObjectNotFound, got %v", err)
	}
}
