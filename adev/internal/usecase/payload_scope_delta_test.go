package usecase

import (
	"encoding/json"
	"testing"
)

// TestEmptyScopeDeltaIsDropped: a FEASIBILITY answer carried a scopeDelta
// record with no added and no removed scope, and the document kept the junk.
func TestEmptyScopeDeltaIsDropped(t *testing.T) {
	payload := json.RawMessage(`{"verdict":"go","scopeDelta":[{"added":"","removed":"","reason":{"text":""}},{"added":"card payment","removed":""}]}`)
	var got struct {
		ScopeDelta []struct {
			Added   string `json:"added"`
			Removed string `json:"removed"`
		} `json:"scopeDelta"`
	}
	if err := json.Unmarshal(dropEmptyScopeDelta(payload), &got); err != nil {
		t.Fatalf("the payload does not decode: %v", err)
	}
	if len(got.ScopeDelta) != 1 || got.ScopeDelta[0].Added != "card payment" {
		t.Fatalf("only the real scope change survives, got %+v", got.ScopeDelta)
	}
}
