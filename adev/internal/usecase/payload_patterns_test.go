package usecase

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Reddetk/Adev/internal/domain"
)

// TestFillPatternFieldsWritesTheChecklist: the model names the deployment pattern,
// the system writes the steps the pattern makes mandatory and keeps every other
// field of the decision.
func TestFillPatternFieldsWritesTheChecklist(t *testing.T) {
	raw := `{"decisions":[
		{"id":"d-1","deployment":"service_deployment_platform","extra":"keep"},
		{"id":"d-2","deployment":"service_per_vm"},
		{"id":"d-3","deployment":"magic"}
	]}`
	got := fillPatternFields(domain.TypeHLD, json.RawMessage(raw))

	var doc struct {
		Decisions []struct {
			ID        string   `json:"id"`
			Checklist []string `json:"checklist"`
			Extra     string   `json:"extra"`
		} `json:"decisions"`
	}
	if err := json.Unmarshal(got, &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(doc.Decisions) != 3 {
		t.Fatalf("want three decisions, got %d", len(doc.Decisions))
	}
	if joined := strings.Join(doc.Decisions[0].Checklist, " | "); !strings.Contains(joined, "disruption budget") {
		t.Fatalf("the platform checklist must reach the payload, got %v", doc.Decisions[0].Checklist)
	}
	if doc.Decisions[0].Extra != "keep" {
		t.Fatalf("an unknown field must survive the rewrite, got %q", doc.Decisions[0].Extra)
	}
	if !strings.Contains(strings.Join(doc.Decisions[1].Checklist, " | "), "systemd") {
		t.Fatalf("the VM checklist must reach the payload, got %v", doc.Decisions[1].Checklist)
	}
	if len(doc.Decisions[2].Checklist) != 0 {
		t.Fatalf("an unknown pattern writes no checklist, got %v", doc.Decisions[2].Checklist)
	}
}

// TestFillPatternFieldsLeavesOtherForms: the LLD has no deployment pattern, so
// the rewrite must not touch it.
func TestFillPatternFieldsLeavesOtherForms(t *testing.T) {
	raw := json.RawMessage(`{"decisions":[{"deployment":"service_per_vm"}]}`)
	if got := fillPatternFields(domain.TypeLLD, raw); string(got) != string(raw) {
		t.Fatalf("another form must pass unchanged, got %s", got)
	}
}
