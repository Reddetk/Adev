package usecase

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Reddetk/Adev/internal/domain"
)

// TestFillTaskRelationsWritesTheRelatedSubset: the plan gets the HLD edges that
// touch a container named by a task, and the unknown fields of the answer survive.
func TestFillTaskRelationsWritesTheRelatedSubset(t *testing.T) {
	hld := json.RawMessage(`{"contextRelations":[
		{"from":"c-1","to":"c-2","description":{"text":"checks the session"},"causeRef":"fr-2"},
		{"from":"c-2","to":"c-3","description":{"text":"shows the progress"},"causeRef":"fr-3"},
		{"from":"c-4","to":"c-5","description":{"text":"passes the grade"},"causeRef":"fr-5"},
		{"from":"c-6","to":"c-7","description":{"text":"opens the lesson"},"causeRef":"fr-6"}
	]}`)
	plan := json.RawMessage(`{"tasks":[
		{"id":"t-1","title":{"text":"Wire the session"},"containerRefs":["c-1","c-2"]},
		{"id":"t-2","title":{"text":"Wire the grade"},"containerRefs":["c-5"]}
	],"contextRelations":[],"extra":"keep"}`)

	got := fillTaskRelations(domain.TypeTaskPlan, plan, hld)

	var doc struct {
		ContextRelations []domain.ContextRelation `json:"contextRelations"`
		Extra            string                   `json:"extra"`
	}
	if err := json.Unmarshal(got, &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if doc.Extra != "keep" {
		t.Fatalf("an unknown field must survive, got %q", doc.Extra)
	}
	if len(doc.ContextRelations) != 3 {
		t.Fatalf("want the three edges touching c-1, c-2 or c-5, got %+v", doc.ContextRelations)
	}
	joined := string(got)
	if !strings.Contains(joined, "fr-2") || !strings.Contains(joined, "fr-5") {
		t.Fatalf("the related requirements must reach the plan: %s", joined)
	}
	if strings.Contains(joined, "fr-6") {
		t.Fatalf("an edge that touches no task container must stay out: %s", joined)
	}
}

// TestFillTaskRelationsLeavesOtherForms: the HLD has no task list, so the rewrite
// must not touch it.
func TestFillTaskRelationsLeavesOtherForms(t *testing.T) {
	payload := json.RawMessage(`{"tasks":[]}`)
	if got := fillTaskRelations(domain.TypeHLD, payload, json.RawMessage(`{}`)); string(got) != string(payload) {
		t.Fatalf("another form must pass unchanged, got %s", got)
	}
}
