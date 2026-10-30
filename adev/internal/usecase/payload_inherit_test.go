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
