package usecase

import (
	"encoding/json"
	"testing"
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
