package usecase

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Reddetk/Adev/internal/domain"
)

// TestHLDCrossFormFailuresReadThePipelineInputs: the stage seam reads the newest
// BRD and SRS of the pipeline, so the HLD cannot invent an actor or an edge that
// the input forms do not hold.
func TestHLDCrossFormFailuresReadThePipelineInputs(t *testing.T) {
	store := newFakeFormStore()
	savePayload(t, store, "1", domain.TypeBRD, domain.BRD{
		Actors: []domain.Actor{{ID: "actor-1", Name: "student"}},
	})
	savePayload(t, store, "1", domain.TypeSRS, domain.SRS{
		FR: []domain.FunctionalRequirement{{
			ID: "fr-1", Actor: "student", Delivery: domain.DeliverySync,
		}},
	})
	stage := NewStageService(NewFormService(store, nil), nil, nil, nil)

	hld := domain.HLD{
		Actors: []domain.HLDActor{{
			ID: "a-student", Name: "student", BrdActorRef: "actor-1",
		}},
		ActorInteractions: []domain.ActorInteraction{{
			ActorID: "a-student", ContainerID: "c-app",
			Description: domain.Text{Value: "enrols in a course"}, CauseRef: "fr-1",
		}},
		Containers: []domain.Container{{
			ObjectID: "c-app", BoundedContextID: "bc-1", Tech: domain.TechGo,
		}},
	}
	payload, err := json.Marshal(hld)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if failures := stage.hldCrossFormFailures(context.Background(), "1", payload); len(failures) != 0 {
		t.Fatalf("a matching HLD must pass the cross-form seam, got %v", failures)
	}

	hld.Actors = nil
	hld.ActorInteractions = nil
	payload, err = json.Marshal(hld)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	failures := stage.hldCrossFormFailures(context.Background(), "1", payload)
	if len(failures) == 0 {
		t.Fatal("an HLD with no C4 person must fail against the BRD actor")
	}
}

func savePayload(t *testing.T, store *fakeFormStore, pipelineID string, formType domain.FormType, payload any) {
	t.Helper()
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal %s: %v", formType, err)
	}
	form := &domain.Form{
		Envelope: domain.Envelope{
			FormID: "FRM-" + pipelineID + "-" + string(formType), FormType: formType,
			SchemaVersion: 1, FormVersion: 1, PipelineID: pipelineID,
		},
		Payload: encoded,
	}
	if err := store.Save(context.Background(), form); err != nil {
		t.Fatalf("save %s: %v", formType, err)
	}
}
