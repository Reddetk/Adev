package usecase

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Reddetk/Adev/internal/domain"
)

// Per-entry retry: a gate failure inside a batched entry is repaired with one more
// call for that entry, not with a regeneration of the whole document.

// TestEntryRepairsMapFailuresToEntries: a failed field inside /fr/3 belongs to the
// entry fr with index 3, and the repair names only that entry.
func TestEntryRepairsMapFailuresToEntries(t *testing.T) {
	payload := json.RawMessage(`{"fr":[
		{"id":"fr-1","statement":{"text":"a"}},
		{"id":"fr-2","statement":{"text":"b"}},
		{"id":"fr-3","statement":{"text":"c"}},
		{"id":"fr-4","statement":{"text":"d"}}
	]}`)
	failures := []domain.ValidationError{
		{Field: "fr[3].acceptance", Problem: "must contain at least one scenario"},
		{Field: "fr[1].actor", Problem: "is empty"},
		{Field: "fr[1].trigger", Problem: "is empty"},
	}
	repairs := entryRepairs(domain.TypeSRS, payload, failures)
	if len(repairs) != 2 {
		t.Fatalf("want two entries to repair, got %+v", repairs)
	}
	if repairs[0].id != "fr-4" || len(repairs[0].fields) != 1 {
		t.Fatalf("want fr-4 with one field, got %+v", repairs[0])
	}
	if repairs[1].id != "fr-2" || len(repairs[1].fields) != 2 {
		t.Fatalf("want fr-2 with two fields, got %+v", repairs[1])
	}
}

// TestEntryRepairsIgnoreTheDocumentLevel: a premise or a verdict is not an entry
// of a list, so the whole-document retry or the operator keeps them.
func TestEntryRepairsIgnoreTheDocumentLevel(t *testing.T) {
	payload := json.RawMessage(`{"premises":{"traffic":{"peakQps":0}},"options":[{"id":"opt-1"}]}`)
	failures := []domain.ValidationError{
		{Field: "premises.traffic.peakQps", Problem: "must be greater than zero"},
		{Field: "verdict", Problem: "unknown value"},
	}
	if repairs := entryRepairs(domain.TypeFeasibility, payload, failures); len(repairs) != 0 {
		t.Fatalf("a document-level field is not an entry, got %+v", repairs)
	}
}

// TestEntryRepairsReachAChildList: a failure inside a nested list repairs the root
// entry that owns it, because the walk calls one entry at a time.
func TestEntryRepairsReachAChildList(t *testing.T) {
	payload := json.RawMessage(`{"containers":[
		{"containerId":"c-1","layers":[]},
		{"containerId":"c-2","layers":[]}
	]}`)
	failures := []domain.ValidationError{
		{Field: "containers[1].layers", Problem: "at least one layer is required"},
	}
	repairs := entryRepairs(domain.TypeLLD, payload, failures)
	if len(repairs) != 1 || repairs[0].id != "c-2" {
		t.Fatalf("want the container c-2 repaired, got %+v", repairs)
	}
}

// TestAnEmptyFieldIsRepairedForItsEntry: the detail call leaves the acceptance of
// one requirement empty. The stage repairs that requirement with one more call
// instead of regenerating the whole SRS or asking the operator about a decision the
// model owes.
func TestAnEmptyFieldIsRepairedForItsEntry(t *testing.T) {
	ctx := context.Background()
	store := newFakeFormStore()
	forms := NewFormService(store, nil)

	detailCalls, repairCalls := 0, 0
	respond := func(request LLMRequest) (LLMResponse, error) {
		switch {
		case strings.Contains(request.User, "The gates rejected these fields of this entry"):
			repairCalls++
			return LLMResponse{Payload: json.RawMessage(`{"entry":{
				"id":"fr-1","actor":"student","trigger":"The student opens the course.",
				"response":"The system records the enrolment.","stateChange":"The student is enrolled.",
				"priority":"must","concurrency":"per_key","idempotency":"none","delivery":"sync","failureMode":"fail_fast",
				"statement":{"text":"The student enrols in the course.","modality":"shall","priority":"must"},
				"acceptance":[{"given":"a student","when":"it enrols","then":"the enrolment exists"}]
			}}`)}, nil
		case strings.Contains(request.User, "the fr entry with id"):
			detailCalls++
			return LLMResponse{Payload: json.RawMessage(`{"entry":{
				"id":"fr-1","actor":"student","trigger":"The student opens the course.",
				"response":"The system records the enrolment.","stateChange":"The student is enrolled.",
				"priority":"must","concurrency":"per_key","idempotency":"none","delivery":"sync","failureMode":"fail_fast",
				"statement":{"text":"The student enrols in the course.","modality":"shall","priority":"must"},
				"acceptance":[]
			}}`)}, nil
		default:
			return LLMResponse{Payload: json.RawMessage(`{"fr":[{"id":"fr-1","statement":{"text":"The student enrols in the course."}}],"nfr":[]}`)}, nil
		}
	}

	client := &calledLLM{next: respond}
	stages := NewStageService(forms, nil, FixedLLM(client), nil)
	stage := domain.Stage{ID: "02", Name: "SRS", Owner: domain.RoleAnalyst,
		Inputs: []domain.FormType{domain.TypeBRD}, Outputs: []domain.FormType{domain.TypeSRS}}

	outcome, err := stages.runForm(
		ctx, StageRequest{PipelineID: "51", TargetSystemID: 3, Route: domain.RouteGreenfield},
		stage, domain.AgentConfig{Role: domain.RoleAnalyst}, client,
		json.RawMessage(`{}`), nil, domain.TypeSRS, "FRM-51-02", 1, nil,
	)
	if err != nil {
		t.Fatalf("run form: %v", err)
	}
	if outcome.Blocked {
		t.Fatalf("the repaired document must pass, gaps: %v", outcome.Gaps)
	}
	if detailCalls != 1 || repairCalls != 1 {
		t.Fatalf("want one detail and one repair call, got %d and %d", detailCalls, repairCalls)
	}
	form, err := forms.Get(ctx, "FRM-51-02", 1)
	if err != nil {
		t.Fatalf("read the form: %v", err)
	}
	if form.Status != domain.StatusValid {
		t.Fatalf("want a valid form, got %s", form.Status)
	}
	var srs domain.SRS
	if err := json.Unmarshal(form.Payload, &srs); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(srs.FR) != 1 || len(srs.FR[0].Acceptance) != 1 {
		t.Fatalf("the repair must fill the acceptance, got %+v", srs.FR)
	}
}

// TestEntryRepairCarriesTheGateReason: an entry can fail on a value that is not
// empty, such as a tech no container runs. The repair call must name the reason the
// gate gave, or the model repeats the same value.
func TestEntryRepairCarriesTheGateReason(t *testing.T) {
	payload := json.RawMessage(`{"decisions":[
		{"id":"d-1","area":"architecture","techs":["java"]},
		{"id":"d-7","area":"storage","techs":["postgresql"]}
	]}`)
	failures := []domain.ValidationError{
		{Field: "decisions[1].techs", Problem: `"postgresql" is not the tech of any container`},
	}
	repairs := entryRepairs(domain.TypeHLD, payload, failures)
	if len(repairs) != 1 {
		t.Fatalf("want one entry to repair, got %+v", repairs)
	}
	if len(repairs[0].problems) != 1 || !strings.Contains(repairs[0].problems[0], "not the tech of any container") {
		t.Fatalf("the repair must carry the gate reason, got %+v", repairs[0].problems)
	}
}
