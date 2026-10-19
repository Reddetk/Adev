package domain

import (
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
		ID: "GAP-1", Pointer: "/payload/metrics", Blocking: true,
		Question: Text{Value: "Which value applies?", Lang: LangSTE100},
	}}

	if err := ValidateBRD(brd, StatusApproved); err == nil {
		t.Fatal("want a blocking gap error, got nil")
	}
	if err := ValidateBRD(brd, StatusValid); err != nil {
		t.Fatalf("a blocking gap must be allowed before approval: %v", err)
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
