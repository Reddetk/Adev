package usecase

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Reddetk/Adev/internal/domain"
)

// TestDetailOfOneContractKeepsTheOtherPaths: the LLD is filled contract by contract.
// A detail call for one contract replaces that contract and its paths, and leaves
// the paths of every other contract as they are.
func TestDetailOfOneContractKeepsTheOtherPaths(t *testing.T) {
	payload := json.RawMessage(`{
		"contracts": [
			{"id": "C-1", "errors": []},
			{"id": "C-2", "errors": []}
		],
		"paths": [
			{"id": "P-1", "contractRef": "C-1", "kind": "happy"},
			{"id": "P-2", "contractRef": "C-2", "kind": "happy"}
		]
	}`)
	detail := json.RawMessage(`{
		"contracts": [{"id": "C-1", "errors": [{"code": "E-1", "condition": "missing"}]}],
		"paths": [
			{"id": "P-3", "contractRef": "C-1", "kind": "negative", "expectedErrorCode": "E-1"}
		]
	}`)

	merged, err := mergeEntry(payload, detail, batchPlan{kind: "contracts", with: "paths", ref: "contractRef"}, "C-1")
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	var doc struct {
		Contracts []struct {
			ID     string            `json:"id"`
			Errors []json.RawMessage `json:"errors"`
		} `json:"contracts"`
		Paths []struct {
			ID          string `json:"id"`
			ContractRef string `json:"contractRef"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(merged, &doc); err != nil {
		t.Fatalf("decode merged: %v", err)
	}
	if len(doc.Contracts) != 2 || len(doc.Contracts[0].Errors) != 1 || len(doc.Contracts[1].Errors) != 0 {
		t.Fatalf("want C-1 detailed and C-2 untouched, got %+v", doc.Contracts)
	}
	got := map[string]string{}
	for _, path := range doc.Paths {
		got[path.ID] = path.ContractRef
	}
	if len(got) != 2 || got["P-2"] != "C-2" || got["P-3"] != "C-1" {
		t.Fatalf("want P-2 kept and P-3 written for C-1, got %+v", doc.Paths)
	}
	if _, stale := got["P-1"]; stale {
		t.Fatal("the old path of C-1 must be replaced")
	}
}

// TestDetailInstructionFillsTheRequiredLayout: a container must get a layer and
// a layer a package; modules stay optional; a contract gets both paths.
func TestDetailInstructionFillsTheRequiredLayout(t *testing.T) {
	stage := domain.Stage{ID: "05"}
	entry := json.RawMessage(`{"name":"app","packages":[]}`)

	container := detailInstruction(domain.TypeLLD, stage, batchPlan{
		kind: "containers", idField: "containerId",
		children: []batchPlan{{kind: "layers", idField: "name"}},
	}, []entryLabel{{plan: batchPlan{kind: "containers", idField: "containerId"}, idValue: "C-1"}}, entry)
	if !strings.Contains(container, "Fill the layers list") {
		t.Fatalf("a container must ask for a layer: %s", container)
	}

	layer := detailInstruction(domain.TypeLLD, stage, batchPlan{
		kind: "layers", idField: "name",
		children: []batchPlan{{kind: "packages", idField: "path"}},
	}, []entryLabel{{plan: batchPlan{kind: "layers", idField: "name"}, idValue: "app"}}, entry)
	if !strings.Contains(layer, "Fill the packages list") {
		t.Fatalf("a layer must ask for a package: %s", layer)
	}

	pkg := detailInstruction(domain.TypeLLD, stage, batchPlan{
		kind: "packages", idField: "path",
		children: []batchPlan{{kind: "modules", idField: "objectId"}},
	}, []entryLabel{{plan: batchPlan{kind: "packages", idField: "path"}, idValue: "internal/app"}}, entry)
	if strings.Contains(pkg, "Fill the modules list") {
		t.Fatalf("modules are optional: %s", pkg)
	}

	contract := detailInstruction(domain.TypeLLD, stage, batchPlan{
		kind: "contracts", with: "paths", ref: "contractRef",
	}, []entryLabel{{plan: batchPlan{kind: "contracts"}, idValue: "C-1"}}, entry)
	if !strings.Contains(contract, "happy path") || !strings.Contains(contract, "moduleRef") {
		t.Fatalf("a contract must ask for both paths and its moduleRef: %s", contract)
	}
}

// TestLLDRetryOnlyForStructuralHoles: the model continues while it can fill the
// layout, but a question or the attempt cap stops the loop.
func TestLLDRetryOnlyForStructuralHoles(t *testing.T) {
	if !lldRetry(domain.TypeLLD, 1, false, false) {
		t.Fatal("a structural hole must get another model pass")
	}
	if lldRetry(domain.TypeLLD, maxLLDFormAttempts, false, false) {
		t.Fatal("the attempt cap must stop the loop")
	}
	if lldRetry(domain.TypeLLD, 1, true, false) {
		t.Fatal("a gap batch waits for the operator")
	}
	if lldRetry(domain.TypeLLD, 1, false, true) {
		t.Fatal("a gate question waits for the operator")
	}
	if lldRetry(domain.TypeBRD, 1, false, false) {
		t.Fatal("only the LLD layout continues by itself")
	}
}

// TestDetailGapsTakeTheEntryPath: a detail answer names a field of its own entry,
// and the walk glues the absolute entry path. An absolute pointer keeps its path,
// and every gap lands in one openGaps list at the document root.
func TestDetailGapsTakeTheEntryPath(t *testing.T) {
	echo := json.RawMessage(`{
		"entry": {"name": "app", "packages": []},
		"openGaps": [
			{"id": "G1", "field": "packages[0].path", "question": {"text": "Which path?", "lang": "STE100"}},
			{"id": "G2", "field": "name", "question": {"text": "Which layer?", "lang": "STE100"}},
			{"id": "G3", "pointer": "/contracts/0/moduleRef", "question": {"text": "Which module?", "lang": "STE100"}}
		]
	}`)

	gaps := detailGaps(echo, "/containers/2/layers/1")
	merged := mergeDetailGaps(json.RawMessage(`{"containers": []}`), gaps)

	var doc struct {
		OpenGaps []struct {
			ID      string `json:"id"`
			Pointer string `json:"pointer"`
		} `json:"openGaps"`
	}
	if err := json.Unmarshal(merged, &doc); err != nil {
		t.Fatalf("decode merged: %v", err)
	}
	if len(doc.OpenGaps) != 3 {
		t.Fatalf("want three gaps, got %d", len(doc.OpenGaps))
	}
	want := []string{
		"/containers/2/layers/1/packages/0/path",
		"/containers/2/layers/1/name",
		"/contracts/0/moduleRef",
	}
	for i, pointer := range want {
		if doc.OpenGaps[i].Pointer != pointer {
			t.Errorf("gap %s: pointer %q, want %q", doc.OpenGaps[i].ID, doc.OpenGaps[i].Pointer, pointer)
		}
	}
}
