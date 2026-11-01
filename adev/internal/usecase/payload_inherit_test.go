package usecase

import (
	"encoding/json"
	"testing"
)

// TestRewriteKeepsTheAcceptedAggregate: stage 04 rewrites the accepted HLD and
// returns an aggregate without root and moduleRef. The accepted values fill the
// hole, the model's own answer wins where it is not empty, and the answered
// questions do not come back.
func TestRewriteKeepsTheAcceptedAggregate(t *testing.T) {
	accepted := json.RawMessage(`{
		"aggregates": [
			{"id": "a-1", "root": "booking", "moduleRef": "module-booking", "invariants": [{"text": "old"}]},
			{"id": "a-2", "root": "tariff", "moduleRef": "module-tariff", "invariants": [{"text": "tariff rule"}]}
		],
		"openGaps": [{"pointer": "/aggregates/0/root"}]
	}`)
	rewrite := json.RawMessage(`{
		"aggregates": [
			{"id": "a-2", "root": "", "invariants": [{"text": "new tariff rule"}]},
			{"id": "a-1", "root": "room", "moduleRef": "module-booking"}
		]
	}`)

	var got struct {
		Aggregates []struct {
			ID         string `json:"id"`
			Root       string `json:"root"`
			ModuleRef  string `json:"moduleRef"`
			Invariants []struct {
				Text string `json:"text"`
			} `json:"invariants"`
		} `json:"aggregates"`
		OpenGaps []json.RawMessage `json:"openGaps"`
	}
	merged := inheritAccepted(accepted, rewrite)
	if err := json.Unmarshal(merged, &got); err != nil {
		t.Fatalf("merged payload does not decode: %v", err)
	}
	if len(got.Aggregates) != 2 {
		t.Fatalf("want two aggregates, got %d", len(got.Aggregates))
	}
	// The list keeps the position of the new answer and takes its values by id.
	byID := map[string]int{}
	for i, aggregate := range got.Aggregates {
		byID[aggregate.ID] = i
	}
	tariff := got.Aggregates[byID["a-2"]]
	if tariff.Root != "tariff" {
		t.Fatalf("the empty root must come from the accepted version, got %q", tariff.Root)
	}
	if tariff.ModuleRef != "module-tariff" {
		t.Fatalf("the missing moduleRef must come from the accepted version, got %q", tariff.ModuleRef)
	}
	if len(tariff.Invariants) != 1 || tariff.Invariants[0].Text != "new tariff rule" {
		t.Fatalf("the model answer must win over the accepted value, got %+v", tariff.Invariants)
	}
	booking := got.Aggregates[byID["a-1"]]
	if booking.Root != "room" {
		t.Fatalf("the model answer must win over the accepted root, got %q", booking.Root)
	}
	if len(booking.Invariants) != 1 || booking.Invariants[0].Text != "old" {
		t.Fatalf("a missing list must come from the accepted version, got %+v", booking.Invariants)
	}
	if len(got.OpenGaps) != 0 {
		t.Fatalf("the answered questions must not come back, got %s", merged)
	}
}

// TestTheOperatorPremisesSurviveARegeneration: a stage re-runs after an answer and
// the model returns the shape with zeros again. The values the operator already
// gave stay: a premise is a fact, not a field the next answer may clear.
func TestTheOperatorPremisesSurviveARegeneration(t *testing.T) {
	draft := json.RawMessage(`{"premises":{"traffic":{"peakQps":50},"storage":{"gbPerYear":200},"team":{"size":5,"languages":["go"],"opsMaturity":"medium"},"budgetMonthly":{"amount":800,"currency":"EUR"}},"options":[]}`)
	next := json.RawMessage(`{"premises":{"traffic":{"peakQps":0},"storage":{"gbPerYear":0},"team":{"size":0,"languages":[],"opsMaturity":""},"budgetMonthly":{"amount":0,"currency":""}},"options":[]}`)

	merged := inheritPremises(draft, next)

	var got struct {
		Premises struct {
			Traffic struct {
				PeakQps int `json:"peakQps"`
			} `json:"traffic"`
			Team struct {
				Size      int      `json:"size"`
				Languages []string `json:"languages"`
			} `json:"team"`
			Budget struct {
				Amount int `json:"amount"`
			} `json:"budgetMonthly"`
		} `json:"premises"`
	}
	if err := json.Unmarshal(merged, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Premises.Traffic.PeakQps != 50 || got.Premises.Team.Size != 5 || got.Premises.Budget.Amount != 800 || len(got.Premises.Team.Languages) != 1 {
		t.Fatalf("the answered premises must survive, got %s", merged)
	}
}

// TestAnAnsweredRequirementSurvivesARegeneration: the SRS is filled FR by FR, and a
// re-run that returns the same ids with empty behaviour fields must keep the
// answers the operator already gave.
func TestAnAnsweredRequirementSurvivesARegeneration(t *testing.T) {
	draft := json.RawMessage(`{"fr":[{"id":"fr-1","concurrency":"per_key","idempotency":"none","delivery":"sync","failureMode":"fail_fast","statement":{"text":"The student enrols."}}]}`)
	next := json.RawMessage(`{"fr":[{"id":"fr-1","concurrency":"","idempotency":"","delivery":"","failureMode":"","statement":{"text":"The student enrols in a course."}}]}`)

	merged := inheritAccepted(draft, next)

	var got struct {
		FR []struct {
			Concurrency string `json:"concurrency"`
			FailureMode string `json:"failureMode"`
		} `json:"fr"`
	}
	if err := json.Unmarshal(merged, &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.FR) != 1 || got.FR[0].Concurrency != "per_key" || got.FR[0].FailureMode != "fail_fast" {
		t.Fatalf("the answered requirement must survive, got %s", merged)
	}
}
