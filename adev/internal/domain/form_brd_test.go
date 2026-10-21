package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestBRDRejectsUnknownMetric(t *testing.T) {
	brd := validBRD()
	brd.Goals[0].MetricID = "M-404"

	err := ValidateBRD(brd, StatusValid)
	if err == nil || !strings.Contains(err.Error(), "metricId") {
		t.Fatalf("want a metricId error, got %v", err)
	}
}

func TestBRDRejectsMetricWithoutUnit(t *testing.T) {
	brd := validBRD()
	brd.Metrics[0].Unit = ""

	err := ValidateBRD(brd, StatusValid)
	if err == nil || !strings.Contains(err.Error(), "unit") {
		t.Fatalf("want a unit error, got %v", err)
	}
}

func TestBRDRejectsDesignTerms(t *testing.T) {
	brd := validBRD()
	brd.Goals[0].Outcome = Text{Value: "The system shall use a module.", Lang: LangSTE100}

	err := ValidateBRD(brd, StatusValid)
	if err == nil || !strings.Contains(err.Error(), "module") {
		t.Fatalf("want a banned term error, got %v", err)
	}
}

func TestBRDRejectsBlockingGapOnApproval(t *testing.T) {
	brd := validBRD()
	brd.OpenGaps = []Gap{{
		ID: "GAP-1", Pointer: "/metrics", Blocking: true,
		Question: Text{Value: "Which value applies?", Lang: LangSTE100},
	}}
	form := newBRDForm(t, brd)

	// Gate 5: a blocking gap is closed before the form is accepted. The shared
	// gate lives in ValidatePayloadWithStatus for every question-phase form.
	form.Status = StatusApproved
	if err := form.ValidatePayloadWithStatus(); err == nil {
		t.Fatal("want a blocking gap error on approved, got nil")
	}
	form.Status = StatusValid
	if err := form.ValidatePayloadWithStatus(); err == nil {
		t.Fatal("want a blocking gap error on valid, got nil")
	}
	form.Status = StatusDraft
	if err := form.ValidatePayloadWithStatus(); err != nil {
		t.Fatalf("a draft may hold a blocking gap: %v", err)
	}

	// A gap that asks the operator holds the acceptance, blocking or not.
	brd.OpenGaps[0].Blocking = false
	form = newBRDForm(t, brd)
	form.Status = StatusValid
	if err := form.ValidatePayloadWithStatus(); err == nil {
		t.Fatal("an unanswered operator question must hold the valid state")
	}

	// A plain gap without a question is only a marked open issue.
	brd.OpenGaps = []Gap{{ID: "GAP-2", Pointer: "/payload/metrics"}}
	form = newBRDForm(t, brd)
	form.Status = StatusValid
	if err := form.ValidatePayloadWithStatus(); err != nil {
		t.Fatalf("a questionless gap may survive the acceptance: %v", err)
	}
}

func TestBRDRejectsAGapPointerOutsideThePayload(t *testing.T) {
	brd := validBRD()
	brd.OpenGaps = []Gap{{
		ID: "GAP-1", Pointer: "/feasibility/costEstimate",
		Question: Text{Value: "What is the cost?", Lang: LangSTE100},
	}}
	form := newBRDForm(t, brd)
	form.Status = StatusDraft

	// The model invented a section: the operator has nowhere to write the answer.
	if err := form.ValidatePayloadWithStatus(); err == nil || !strings.Contains(err.Error(), "/feasibility/costEstimate") {
		t.Fatalf("want a pointer error, got %v", err)
	}

	brd.OpenGaps[0].Pointer = "/metrics/0/baseline"
	form = newBRDForm(t, brd)
	form.Status = StatusDraft
	if err := form.ValidatePayloadWithStatus(); err != nil {
		t.Fatalf("a pointer to a real field must pass: %v", err)
	}

	// A word where the array takes an index names no slot: the answer would fail
	// on arrival, so the question must never reach the operator.
	brd.OpenGaps[0].Pointer = "/metrics/baseline"
	form = newBRDForm(t, brd)
	form.Status = StatusDraft
	if err := form.ValidatePayloadWithStatus(); err == nil {
		t.Fatal("a pointer that the payload cannot take must fail the gate")
	}
}

func TestBRDAcceptsValidBRD(t *testing.T) {
	if err := ValidateBRD(validBRD(), StatusValid); err != nil {
		t.Fatalf("valid brd rejected: %v", err)
	}
}

func TestPayloadTextsOfBRD(t *testing.T) {
	form := newBRDForm(t, validBRD())

	texts, err := form.PayloadTexts()
	if err != nil {
		t.Fatalf("payload texts: %v", err)
	}
	if len(texts) == 0 {
		t.Fatal("want at least one text")
	}
}

func TestValidatePayloadDispatchesByFormType(t *testing.T) {
	form := newBRDForm(t, validBRD())
	if err := form.ValidatePayload(); err != nil {
		t.Fatalf("brd payload rejected: %v", err)
	}
}

// A payload of the wrong shape is a validation failure, not a server error: the
// stage runner feeds it back to the model, which can repair it.
func TestValidatePayloadReportsAMismatchedShapeAsValidationError(t *testing.T) {
	form := &Form{Envelope: Envelope{FormType: TypeBRD, Status: StatusDraft},
		Payload: json.RawMessage(`{"goals":["a branch library"]}`)}

	err := form.ValidatePayload()

	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("want a ValidationError, got %v", err)
	}
	if validation.Field != "payload" {
		t.Fatalf("want field payload, got %q", validation.Field)
	}
	if !strings.Contains(validation.Problem, "goals") {
		t.Fatalf("the problem must name the broken field, got %q", validation.Problem)
	}
}
