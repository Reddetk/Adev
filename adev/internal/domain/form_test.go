package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func ptr(v float64) *float64 { return &v }

func validBRD() BRD {
	return BRD{
		Goals: []Goal{{
			ID:       "GOAL-1",
			Outcome:  Text{Value: "The system shall reduce the cycle time.", Lang: LangSTE100},
			MetricID: "M-1",
		}},
		Metrics: []Metric{{
			ID: "M-1", Name: "cycle_time_hours", Unit: "hour", Baseline: ptr(40), Target: ptr(24),
		}},
		Scope: Scope{
			In:  []Text{{Value: "The system shall plan the work.", Lang: LangSTE100}},
			Out: []Text{{Value: "The system shall host the result.", Lang: LangSTE100}},
		},
		Actors: []Actor{{
			ID: "A-1", Name: "operator",
			Tasks:          []Text{{Value: "The operator sends a request.", Lang: LangSTE100}},
			Frequency:      FrequencyDay,
			BusinessObject: "request",
		}},
		SuccessMetrics: []Success{{MetricID: "M-1", Threshold: "24", Window: "30 day"}},
	}
}

func newBRDForm(t *testing.T, brd BRD) *Form {
	t.Helper()
	form := &Form{Envelope: Envelope{
		FormID: "FRM-1", FormType: TypeBRD, SchemaVersion: 1, FormVersion: 1,
		Status: StatusValid, Route: RouteGreenfield, Stage: "00", Mode: ModeFull,
		PipelineID: "PIPE-1", TargetSystemID: "TGT-1", ProducedBy: RoleAnalyst,
		Inputs: []Ref{{FormID: "FRM-0", FormType: TypeUserInput, FormVersion: 1, Hash: "sha256:x"}},
	}}
	if err := form.SetPayload(brd); err != nil {
		t.Fatalf("set payload: %v", err)
	}
	return form
}

func TestHashIsOrderIndependent(t *testing.T) {
	left := json.RawMessage(`{"b":1,"a":{"d":4,"c":3}}`)
	right := json.RawMessage(`{"a":{"c":3,"d":4},"b":1}`)

	hashLeft, err := HashPayload(left)
	if err != nil {
		t.Fatalf("hash left: %v", err)
	}
	hashRight, err := HashPayload(right)
	if err != nil {
		t.Fatalf("hash right: %v", err)
	}
	if hashLeft != hashRight {
		t.Fatalf("hash depends on key order: %s != %s", hashLeft, hashRight)
	}
	if !strings.HasPrefix(hashLeft, "sha256:") {
		t.Fatalf("unexpected hash format: %s", hashLeft)
	}
}

func TestEnvelopeRejectsDeltaWithoutBase(t *testing.T) {
	form := newBRDForm(t, validBRD())
	form.Mode = ModeDelta
	form.Ops = json.RawMessage(`[]`)

	err := form.ValidateEnvelope(nil)
	if err == nil || !strings.Contains(err.Error(), "base") {
		t.Fatalf("want a base error, got %v", err)
	}
}

func TestEnvelopeRejectsHashMismatch(t *testing.T) {
	form := newBRDForm(t, validBRD())
	form.PayloadHash = "sha256:deadbeef"

	err := form.ValidateEnvelope(nil)
	if err == nil || !strings.Contains(err.Error(), "payloadHash") {
		t.Fatalf("want a payloadHash error, got %v", err)
	}
}

func TestEnvelopeRejectsVersionWithoutPrevious(t *testing.T) {
	form := newBRDForm(t, validBRD())
	form.FormVersion = 2

	err := form.ValidateEnvelope(nil)
	if err == nil || !strings.Contains(err.Error(), "previousFormVersion") {
		t.Fatalf("want a previousFormVersion error, got %v", err)
	}
}

func TestEnvelopeAcceptsResolvedInputs(t *testing.T) {
	form := newBRDForm(t, validBRD())
	resolved := []ResolvedRef{{Ref: form.Inputs[0], PayloadHash: "sha256:x"}}

	if err := form.ValidateEnvelope(resolved); err != nil {
		t.Fatalf("valid form rejected: %v", err)
	}
}

func TestEnvelopeRejectsUnresolvedInput(t *testing.T) {
	form := newBRDForm(t, validBRD())

	err := form.ValidateEnvelope(nil)
	if err == nil || !strings.Contains(err.Error(), "not resolved") {
		t.Fatalf("want an unresolved input error, got %v", err)
	}
}

func TestProjectionHidesPayloadFromDev(t *testing.T) {
	form := newBRDForm(t, validBRD())

	projected, err := BRDProjection.Project(RoleDev, form)
	if err != nil {
		t.Fatalf("project: %v", err)
	}
	fields, ok := projected.(map[string]any)
	if !ok {
		t.Fatalf("unexpected projection type %T", projected)
	}
	if _, exists := fields["payload"]; exists {
		t.Fatal("dev must not see the payload")
	}
	if fields["formId"] != "FRM-1" {
		t.Fatalf("dev lost formId: %v", fields)
	}
}

func TestProjectionKeepsPayloadForAnalyst(t *testing.T) {
	form := newBRDForm(t, validBRD())

	projected, err := BRDProjection.Project(RoleAnalyst, form)
	if err != nil {
		t.Fatalf("project: %v", err)
	}
	fields, ok := projected.(map[string]any)
	if !ok {
		t.Fatalf("unexpected projection type %T", projected)
	}
	payload, ok := fields["payload"].(map[string]any)
	if !ok {
		t.Fatalf("analyst lost the payload: %v", fields)
	}
	if _, exists := payload["goals"]; !exists {
		t.Fatalf("analyst lost goals: %v", payload)
	}
}

func TestProjectionOperatorSeesEverything(t *testing.T) {
	form := newBRDForm(t, validBRD())

	projected, err := BRDProjection.Project(RoleOperator, form)
	if err != nil {
		t.Fatalf("project: %v", err)
	}
	fields, ok := projected.(map[string]any)
	if !ok {
		t.Fatalf("unexpected projection type %T", projected)
	}
	if _, exists := fields["payload"]; !exists {
		t.Fatalf("operator lost the payload: %v", fields)
	}
}
