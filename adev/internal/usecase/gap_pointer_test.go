package usecase

import (
	"encoding/json"
	"testing"
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
