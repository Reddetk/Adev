package usecase

import (
	"encoding/json"
	"testing"

	"github.com/Reddetk/Adev/internal/domain"
)

func TestToJSONPointer(t *testing.T) {
	cases := map[string]string{
		"scope.in[0]":            "/scope/in/0",
		"constraints[3]":         "/constraints/3",
		"actors.0.tasks[2].text": "/actors/0/tasks/2/text",
		"$.metrics[0].baseline":  "/metrics/0/baseline",
		"$.scope.out":            "/scope/out",
		"":                       "",
	}
	for raw, want := range cases {
		if got := toJSONPointer(raw); got != want {
			t.Errorf("toJSONPointer(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestNormalizeGapPointersPutsAFieldOfAnEmptyListAtIndexZero(t *testing.T) {
	in := json.RawMessage(`{"goals":[],"openGaps":[
		{"id":"GAP-1","pointer":"/goals/outcome","question":{"text":"q1"}},
		{"id":"GAP-2","pointer":"/metrics/name","question":{"text":"q2"}}]}`)
	out := normalizeGapPointers(in)

	var doc struct {
		Gaps []struct {
			Pointer string `json:"pointer"`
		} `json:"openGaps"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if doc.Gaps[0].Pointer != "/goals/0/outcome" {
		t.Errorf("field of an empty list not put at index 0: %q", doc.Gaps[0].Pointer)
	}
	if doc.Gaps[1].Pointer != "/metrics/name" {
		t.Errorf("a missing list must keep the pointer: %q", doc.Gaps[1].Pointer)
	}
}

func TestNormalizeGapPointers(t *testing.T) {
	in := json.RawMessage(`{"goals":[{"id":"g1"}],"openGaps":[
		{"id":"GAP-1","pointer":"scope.in[0]","question":{"text":"q1"}},
		{"id":"GAP-2","pointer":"/constraints/2","question":{"text":"q2"}},
		{"id":"GAP-3","pointer":"/payload/metrics/0/baseline","question":{"text":"q3"}}]}`)
	out := normalizeGapPointers(in)

	var doc struct {
		Goals []map[string]any `json:"goals"`
		Gaps  []struct {
			ID      string `json:"id"`
			Pointer string `json:"pointer"`
		} `json:"openGaps"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(doc.Goals) != 1 || doc.Goals[0]["id"] != "g1" {
		t.Errorf("other sections damaged: %s", out)
	}
	if doc.Gaps[0].Pointer != "/scope/in/0" {
		t.Errorf("dot pointer not normalized: %q", doc.Gaps[0].Pointer)
	}
	if doc.Gaps[1].Pointer != "/constraints/2" {
		t.Errorf("valid pointer rewritten: %q", doc.Gaps[1].Pointer)
	}
	if doc.Gaps[2].Pointer != "/metrics/0/baseline" {
		t.Errorf("root /payload segment not dropped: %q", doc.Gaps[2].Pointer)
	}
}

// TestPurgeKeepsOnlyValueGaps: the operator answers one value. A gap that names a
// whole list, a layer record or a layout field is the next run's job, so the purge
// leaves only the scalar field outside the layout. The LLD gaps reader has to see
// the payload for the purge to work at all.
func TestPurgeKeepsOnlyValueGaps(t *testing.T) {
	payload := json.RawMessage(`{
		"containers": [{"containerId": "C-1", "layers": [{"name": "app"}]}],
		"contracts": [{"id": "C-1", "errors": []}], "paths": [],
		"openGaps": [
			{"id": "G1", "pointer": "/containers/0/layers/0/name", "question": {"text": "Which layer?", "lang": "STE100"}},
			{"id": "G2", "pointer": "/containers/0/layers", "question": {"text": "Which layers?", "lang": "STE100"}},
			{"id": "G3", "pointer": "/containers/0/layers/0", "question": {"text": "Describe the layer", "lang": "STE100"}},
			{"id": "G4", "pointer": "/paths", "question": {"text": "Which paths?", "lang": "STE100"}},
			{"id": "G5", "pointer": "/contracts/0/errors/0/condition", "question": {"text": "Which failure?", "lang": "STE100"}}
		]
	}`)

	purged := purgeUnaskableGaps(domain.TypeLLD, payload)

	var doc struct {
		Gaps []struct {
			ID string `json:"id"`
		} `json:"openGaps"`
	}
	if err := json.Unmarshal(purged, &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(doc.Gaps) != 1 || doc.Gaps[0].ID != "G5" {
		t.Fatalf("want only the contract value gap G5, got %+v", doc.Gaps)
	}
}

// TestStructuralGateFailureAsksNothing: a gate finding about a whole list or the
// layout is the next run's job. Only the finding that names one value outside the
// layout becomes a question, or the operator receives a card no answer can close.
func TestStructuralGateFailureAsksNothing(t *testing.T) {
	payload := json.RawMessage(`{"containers":[{"containerId":"C-1","layers":[{"name":"app"}]}],"contracts":[{"id":"C-1","errors":[]}],"paths":[]}`)
	failures := []domain.ValidationError{
		{Field: "containers[0].layers", Problem: "at least one layer is required"},
		{Field: "containers[0].layers[0].packages", Problem: "at least one package is required"},
		{Field: "containers[0].layers[0].name", Problem: `unknown value ""`},
		{Field: "contracts[0].errors[0].condition", Problem: "is empty"},
	}

	got := valueFailures(domain.TypeLLD, payload, failures)

	if len(got) != 1 || got[0].Field != "contracts[0].errors[0].condition" {
		t.Fatalf("want only the contract value finding, got %+v", got)
	}
}
