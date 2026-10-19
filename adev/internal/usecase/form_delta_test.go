package usecase

import (
	"encoding/json"
	"testing"
)

func TestApplyDelta(t *testing.T) {
	base := json.RawMessage(`{"fr":[{"id":"FR-1","priority":"must"}],"outOfScope":[]}`)
	ops := json.RawMessage(`[
		{"op":"replace","path":"/fr/0/priority","value":"high"},
		{"op":"add","path":"/fr/-","value":{"id":"FR-2","priority":"low"}},
		{"op":"remove","path":"/outOfScope"}
	]`)

	result, err := ApplyDelta(base, ops)
	if err != nil {
		t.Fatalf("apply delta: %v", err)
	}

	var decoded struct {
		FR []struct {
			ID       string `json:"id"`
			Priority string `json:"priority"`
		} `json:"fr"`
		OutOfScope json.RawMessage `json:"outOfScope"`
	}
	if err := json.Unmarshal(result, &decoded); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if len(decoded.FR) != 2 || decoded.FR[0].Priority != "high" || decoded.FR[1].ID != "FR-2" {
		t.Fatalf("unexpected delta result: %s", result)
	}
	if decoded.OutOfScope != nil {
		t.Fatalf("removed field is still present: %s", result)
	}
}

func TestApplyDeltaRejectsBrokenPatch(t *testing.T) {
	base := json.RawMessage(`{"a":1}`)
	if _, err := ApplyDelta(base, json.RawMessage(`[{"op":"replace","path":"/missing","value":2}]`)); err == nil {
		t.Fatal("want an error for a patch against a missing path")
	}
}
