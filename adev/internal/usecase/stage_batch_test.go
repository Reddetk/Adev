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

// TestAssignIDsFillsTheMissingIdentifier: the outline may arrive without ids. The
// id is structural, so the system names the entry itself, keeps the ids the model
// did give, and never collides with one of them. A list keyed by a business name
// keeps its own keys.
func TestAssignIDsFillsTheMissingIdentifier(t *testing.T) {
	entries := []json.RawMessage{
		json.RawMessage(`{"priority":"should"}`),
		json.RawMessage(`{"id":"FR-1"}`),
		json.RawMessage(`{"statement":{"text":"x"}}`),
	}
	got := assignIDs(entries, batchPlan{kind: "fr"})
	ids := []string{fieldOf(got[0], "id"), fieldOf(got[1], "id"), fieldOf(got[2], "id")}
	want := []string{"FR-1-2", "FR-1", "FR-3"}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("entry %d: id %q, want %q (all: %v)", i, ids[i], want[i], ids)
		}
	}

	named := []json.RawMessage{json.RawMessage(`{"packages":[]}`)}
	if out := assignIDs(named, batchPlan{kind: "layers", idField: "name"}); fieldOf(out[0], "name") != "" {
		t.Fatal("a layer must keep the model's name, not get a generated one")
	}
}

// TestMergeEntryFieldsKeepsWhatTheSystemFilled: the model writes the meaning of
// the entry; the keys and the fields it omits keep the value of the skeleton, and
// a paragraph never replaces the entry.
func TestMergeEntryFieldsKeepsWhatTheSystemFilled(t *testing.T) {
	existing := json.RawMessage(`{"containerId":"C-1","tech":"go","purpose":{"text":"old"}}`)
	updated := json.RawMessage(`{"tech":"rust","purpose":{"text":"new"}}`)

	merged := mergeEntryFields(existing, updated)
	var doc map[string]any
	if err := json.Unmarshal(merged, &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if doc["containerId"] != "C-1" {
		t.Fatalf("the key of the document must survive, got %v", doc["containerId"])
	}
	if doc["tech"] != "rust" {
		t.Fatalf("the answer must update a field it names, got %v", doc["tech"])
	}

	prose := mergeEntryFields(existing, json.RawMessage(`"Контракт для получения тарифа"`))
	if string(prose) != string(existing) {
		t.Fatalf("a paragraph must not replace the entry, got %s", prose)
	}
}

// TestWithEntryKeysFillsWhatTheSystemOwns: the model answers an object without
// the ids; the merge needs the key and the companion reference to find the entry.
func TestWithEntryKeysFillsWhatTheSystemOwns(t *testing.T) {
	echo := json.RawMessage(`{
		"entry": {"inputType": "CreateTariffRequest", "outputType": "Tariff"},
		"paths": [
			{"kind": "happy", "given": "a", "when": "b", "then": "c"},
			{"kind": "negative", "given": "a", "when": "b", "then": "c", "expectedErrorCode": "E-1"}
		]}`)
	plan := batchPlan{kind: "contracts", with: "paths", ref: "contractRef"}
	labels := []entryLabel{{plan: batchPlan{kind: "contracts"}, idValue: "contract-tariff-create"}}

	got := withEntryKeys(echo, plan, labels)

	var doc struct {
		Entry struct {
			ID string `json:"id"`
		} `json:"entry"`
		Paths []struct {
			ID          string `json:"id"`
			ContractRef string `json:"contractRef"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(got, &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if doc.Entry.ID != "contract-tariff-create" {
		t.Fatalf("the entry must take the key of the walk, got %q", doc.Entry.ID)
	}
	if len(doc.Paths) != 2 || doc.Paths[0].ContractRef != "contract-tariff-create" {
		t.Fatalf("every path must point at the contract, got %+v", doc.Paths)
	}
	if doc.Paths[0].ID == "" || doc.Paths[1].ID == "" {
		t.Fatalf("the paths need system ids, got %+v", doc.Paths)
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
	if !strings.Contains(pkg, "Fill the path of this package") {
		t.Fatalf("a package must get a path: %s", pkg)
	}

	contract := detailInstruction(domain.TypeLLD, stage, batchPlan{
		kind: "contracts", with: "paths", ref: "contractRef",
	}, []entryLabel{{plan: batchPlan{kind: "contracts"}, idValue: "C-1"}}, entry)
	if !strings.Contains(contract, "happy path") || !strings.Contains(contract, "moduleRef") {
		t.Fatalf("a contract must ask for both paths and its moduleRef: %s", contract)
	}
}

// TestStructuralRetryOnlyForDesignHoles: a design document continues while it can
// fill the layout, and the attempt cap stops the loop.
func TestStructuralRetryOnlyForDesignHoles(t *testing.T) {
	for _, output := range []domain.FormType{domain.TypeLLD, domain.TypeHLD} {
		if !structuralRetry(output, 1) {
			t.Fatalf("%s: a structural hole must get another model pass", output)
		}
		if structuralRetry(output, maxLLDFormAttempts) {
			t.Fatalf("%s: the attempt cap must stop the loop", output)
		}
	}
	for _, output := range []domain.FormType{domain.TypeBRD, domain.TypeSRS} {
		if structuralRetry(output, 1) {
			t.Fatalf("%s: only a design layout continues by itself", output)
		}
	}
}

// TestSkeletonInstructionKeepsTheChildListsEmpty: the first LLD pass is the tree
// outline only. Every child list waits for its own pass, so one answer never
// carries the whole document.
func TestSkeletonInstructionKeepsTheChildListsEmpty(t *testing.T) {
	got := skeletonInstruction(domain.Stage{ID: "05"}, domain.TypeLLD)

	for _, want := range []string{"Leave the detailed lists empty", "layers", "packages", "modules"} {
		if !strings.Contains(got, want) {
			t.Fatalf("the skeleton must ask for the outline only, missing %q: %s", want, got)
		}
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
