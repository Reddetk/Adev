package usecase

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Reddetk/Adev/internal/domain"
)

// TestProposalsOfExtractsTheLayoutObjects: an LLD names the layers, packages and
// modules of every container, each with the container it belongs to.
func TestProposalsOfExtractsTheLayoutObjects(t *testing.T) {
	stages := NewStageService(nil, nil, nil, nil)

	raw := mustJSON(t, map[string]any{
		"containers": []map[string]any{{
			"containerId": "obj-container-1",
			"layers": []map[string]any{{
				"name": "app",
				"packages": []map[string]any{{
					"path":    "internal/app",
					"modules": []map[string]any{{"objectId": "obj-module-1", "path": "internal/app/book"}},
				}},
			}},
		}},
	})
	lld := &domain.Form{Envelope: domain.Envelope{FormID: "FRM-1-05", FormType: domain.TypeLLD, FormVersion: 1}}
	if err := lld.SetRawPayload(raw); err != nil {
		t.Fatalf("lld payload: %v", err)
	}

	objects, err := stages.proposalsOf(lld)
	if err != nil {
		t.Fatalf("proposals of lld: %v", err)
	}
	if len(objects) != 3 { // layer, package, module
		t.Fatalf("want 3 proposals, got %d: %+v", len(objects), objects)
	}
	if layer := objects[0]; layer.Kind != domain.KindLayer || layer.Name != "app" || layer.ContainerID != "obj-container-1" {
		t.Fatalf("bad layer proposal: %+v", layer)
	}
	if pkg := objects[1]; pkg.Kind != domain.KindPackage || pkg.Name != "internal/app" || pkg.Path != "internal/app" || pkg.ContainerID != "obj-container-1" {
		t.Fatalf("bad package proposal: %+v", pkg)
	}
	if module := objects[2]; module.Kind != domain.KindModule || module.Name != "obj-module-1" || module.Path != "internal/app/book" {
		t.Fatalf("bad module proposal: %+v", module)
	}
}

// TestProposalsOfNamesTheContainersOfTheHLD.
func TestProposalsOfNamesTheContainers(t *testing.T) {
	stages := NewStageService(nil, nil, nil, nil)

	hld := &domain.Form{Envelope: domain.Envelope{FormID: "FRM-1-03", FormType: domain.TypeHLD, FormVersion: 1}}
	if err := hld.SetRawPayload(mustJSON(t, map[string]any{
		"containers": []map[string]any{{
			"objectId":         "obj-container-1",
			"purpose":          map[string]any{"text": "Serve the lending desk.", "lang": "ste100"},
			"boundedContextId": "bc-1",
			"tech":             "go",
		}},
	})); err != nil {
		t.Fatalf("hld payload: %v", err)
	}

	objects, err := stages.proposalsOf(hld)
	if err != nil {
		t.Fatalf("proposals of hld: %v", err)
	}
	if len(objects) != 1 || objects[0].Kind != domain.KindContainer || objects[0].Name != "obj-container-1" {
		t.Fatalf("bad container proposals: %+v", objects)
	}
	if objects[0].Fields["tech"] != "go" || objects[0].Fields["boundedContextId"] != "bc-1" {
		t.Fatalf("bad container fields: %+v", objects[0].Fields)
	}
}

// TestRegisterObjectsIsIdempotent: the same document registers its objects
// once; a second pass of the same proposals changes nothing.
func TestRegisterObjectsIsIdempotent(t *testing.T) {
	store := newFakeObjectStore()
	stages := NewStageService(nil, nil, nil, nil).WithRegistry(NewRegistryService(store))

	form := &domain.Form{Envelope: domain.Envelope{
		FormID: "FRM-1-03", FormType: domain.TypeHLD, FormVersion: 1,
		TargetSystemID: "1", Stage: "03",
	}}
	if err := form.SetRawPayload(mustJSON(t, map[string]any{
		"containers": []map[string]any{{
			"objectId":         "obj-container-1",
			"purpose":          map[string]any{"text": "Serve the lending desk.", "lang": "ste100"},
			"boundedContextId": "bc-1",
			"tech":             "go",
		}},
	})); err != nil {
		t.Fatalf("hld payload: %v", err)
	}
	request := StageRequest{PipelineID: "1", TargetSystemID: 1}

	ctx := context.Background()
	stages.registerObjects(ctx, request, form)
	stages.registerObjects(ctx, request, form)

	objects, err := store.FindByTargetSystem(ctx, 1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(objects) != 1 {
		t.Fatalf("want 1 registered container, got %d: %+v", len(objects), objects)
	}
	if objects[0].Status != domain.ObjectPlanned || objects[0].Name != "obj-container-1" {
		t.Fatalf("bad object: %+v", objects[0])
	}
	if introduced := objects[0].IntroducedBy; introduced.FormID != "FRM-1-03" || introduced.FormVersion != 1 {
		t.Fatalf("the object must name the form that introduced it: %+v", introduced)
	}
}

func mustJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return raw
}
