package usecase

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Reddetk/Adev/internal/domain"
)

// lldInputs is the stage context of the LLD: the HLD branch of one contract and
// one decision of that branch.
const lldInputs = `{
	"FRM-1-03": {
		"formId": "FRM-1-03", "formType": "HLD", "formVersion": 1,
		"payload": {
			"boundedContexts": [
				{"id": "BC-CHG", "containerIds": ["CNT-CHG"]},
				{"id": "BC-STN", "containerIds": ["CNT-STN"]},
				{"id": "BC-BILL", "containerIds": ["CNT-BILL"]}
			],
			"containers": [
				{"objectId": "CNT-CHG", "boundedContextId": "BC-CHG"},
				{"objectId": "CNT-STN", "boundedContextId": "BC-STN"},
				{"objectId": "CNT-BILL", "boundedContextId": "BC-BILL"}
			],
			"aggregates": [
				{"id": "AGG-CHG", "containerId": "CNT-CHG", "invariants": [{"text": "one session per connector"}]},
				{"id": "AGG-STN", "containerId": "CNT-STN", "invariants": [{"text": "one session at a time"}]},
				{"id": "AGG-BILL", "containerId": "CNT-BILL", "invariants": [{"text": "one receipt"}]}
			],
			"contextRelations": [
				{"contractId": "CTR-CONNECTOR-STATE", "from": "BC-CHG", "to": "BC-STN", "type": "sync"},
				{"contractId": "EVT-SESSION-COMPLETED", "from": "BC-CHG", "to": "BC-BILL", "type": "async"}
			],
			"ubiquitousLanguage": [{"term": "Connector", "definition": {"text": "A socket of a station."}}],
			"degradationStrategy": [{"failure": "The station service does not answer"}]
		}
	},
	"FRM-1-04-ADR": {
		"formId": "FRM-1-04-ADR", "formType": "ADR", "formVersion": 1,
		"payload": {
			"decisions": [
				{"id": "ADR-001", "objectRefs": ["CNT-CHG", "CNT-STN"]},
				{"id": "ADR-002", "objectRefs": ["CNT-BILL"]},
				{"id": "ADR-003"}
			]
		}
	}
}`

// hldInputs is the stage context of the HLD: the SRS, the previous HLD and the
// decisions. A container call reads the branch of its own container.
const hldInputs = `{
	"FRM-1-02": {
		"formId": "FRM-1-02", "formType": "SRS", "formVersion": 1,
		"payload": {
			"fr": [{"id": "FR-1"}, {"id": "FR-2"}],
			"nfr": [{"id": "NFR-1"}, {"id": "NFR-2"}]
		}
	},
	"FRM-1-03": {
		"formId": "FRM-1-03", "formType": "HLD", "formVersion": 1,
		"payload": {
			"boundedContexts": [
				{"id": "BC-CHG", "containerIds": ["CNT-CHG"]},
				{"id": "BC-BILL", "containerIds": ["CNT-BILL"]}
			],
			"containers": [
				{"objectId": "CNT-CHG", "boundedContextId": "BC-CHG", "nfrRefs": ["NFR-1"]},
				{"objectId": "CNT-BILL", "boundedContextId": "BC-BILL", "nfrRefs": ["NFR-2"]}
			],
			"aggregates": [
				{"id": "AGG-CHG", "containerId": "CNT-CHG"},
				{"id": "AGG-BILL", "containerId": "CNT-BILL"}
			],
			"contextRelations": [
				{"from": "BC-CHG", "to": "BC-BILL"},
				{"from": "BC-BILL", "to": "BC-OTHER"}
			]
		}
	},
	"FRM-1-04-ADR": {
		"formId": "FRM-1-04-ADR", "formType": "ADR", "formVersion": 1,
		"payload": {
			"decisions": [
				{"id": "ADR-1", "objectRefs": ["CNT-CHG"]},
				{"id": "ADR-2", "objectRefs": ["CNT-BILL"]}
			]
		}
	}
}`

// TestFocusContextKeepsTheContractBranch: a contract call reads its two contexts,
// their containers, their aggregates and the decisions that name them, not the
// containers of the other contexts.
func TestFocusContextKeepsTheContractBranch(t *testing.T) {
	labels := []entryLabel{{plan: batchPlan{kind: "contracts"}, idValue: "CTR-CONNECTOR-STATE"}}
	focused := focusContext(domain.TypeLLD, json.RawMessage(lldInputs), labels, nil)

	var forms map[string]struct {
		Payload struct {
			BoundedContexts  []struct{ ID string }
			Containers       []struct{ ObjectID string }
			Aggregates       []struct{ ID string }
			ContextRelations []struct{ ContractID string }
			Decisions        []struct{ ID string }
		}
	}
	if err := json.Unmarshal(focused, &forms); err != nil {
		t.Fatalf("decode focused: %v", err)
	}
	hld := forms["FRM-1-03"].Payload
	if len(hld.Containers) != 2 || hld.Containers[0].ObjectID == "CNT-BILL" || hld.Containers[1].ObjectID == "CNT-BILL" {
		t.Fatalf("want the two containers of the branch, got %+v", hld.Containers)
	}
	if len(hld.Aggregates) != 2 {
		t.Fatalf("want the two aggregates of the branch, got %d", len(hld.Aggregates))
	}
	if len(hld.BoundedContexts) != 2 {
		t.Fatalf("want the two contexts of the branch, got %d", len(hld.BoundedContexts))
	}
	if len(hld.ContextRelations) != 1 || hld.ContextRelations[0].ContractID != "CTR-CONNECTOR-STATE" {
		t.Fatalf("want the relation of the contract, got %+v", hld.ContextRelations)
	}
	decisions := forms["FRM-1-04-ADR"].Payload.Decisions
	if len(decisions) != 2 {
		t.Fatalf("want the branch decisions and the global one, got %+v", decisions)
	}
	if strings.Contains(string(focused), "AGG-BILL") {
		t.Fatal("the aggregate of another context must not reach the call")
	}
}

// TestFocusContextKeepsTheContainerBranch: a layer or package call stays inside
// the container and its context.
func TestFocusContextKeepsTheContainerBranch(t *testing.T) {
	labels := []entryLabel{
		{plan: batchPlan{kind: "containers", idField: "containerId"}, idValue: "CNT-STN"},
		{plan: batchPlan{kind: "layers", idField: "name"}, idValue: "app"},
	}
	focused := focusContext(domain.TypeLLD, json.RawMessage(lldInputs), labels, nil)
	var forms map[string]struct {
		Payload struct {
			Containers      []struct{ ObjectID string }
			Aggregates      []struct{ ID string }
			BoundedContexts []struct{ ID string }
		}
	}
	if err := json.Unmarshal(focused, &forms); err != nil {
		t.Fatalf("decode focused: %v", err)
	}
	payload := forms["FRM-1-03"].Payload
	if len(payload.Containers) != 1 || payload.Containers[0].ObjectID != "CNT-STN" {
		t.Fatalf("a layer call reads its own container only, got %+v", payload.Containers)
	}
	if len(payload.Aggregates) != 1 || payload.Aggregates[0].ID != "AGG-STN" {
		t.Fatalf("a layer call reads the aggregates of its container only, got %+v", payload.Aggregates)
	}
	if len(payload.BoundedContexts) != 1 || payload.BoundedContexts[0].ID != "BC-STN" {
		t.Fatalf("a layer call reads its own context only, got %+v", payload.BoundedContexts)
	}
}

// TestFocusContextKeepsTheHLDContainerBranch: a container call reads its own
// container, its aggregates, its bounded context, the relations of that context,
// the NFRs it names and the decisions that mention it. The functional
// requirements have no container link, so they all stay.
func TestFocusContextKeepsTheHLDContainerBranch(t *testing.T) {
	labels := []entryLabel{{plan: batchPlan{kind: "containers", idField: "objectId"}, idValue: "CNT-CHG"}}
	entry := json.RawMessage(`{"objectId":"CNT-CHG","boundedContextId":"BC-CHG","nfrRefs":["NFR-1"]}`)
	focused := focusContext(domain.TypeHLD, json.RawMessage(hldInputs), labels, entry)

	var forms map[string]struct {
		Payload struct {
			FR               []struct{ ID string }
			NFR              []struct{ ID string }
			BoundedContexts  []struct{ ID string }
			Containers       []struct{ ObjectID string }
			Aggregates       []struct{ ID string }
			ContextRelations []struct{ From, To string }
			Decisions        []struct{ ID string }
		}
	}
	if err := json.Unmarshal(focused, &forms); err != nil {
		t.Fatalf("decode focused: %v", err)
	}
	hld := forms["FRM-1-03"].Payload
	if len(hld.Containers) != 1 || hld.Containers[0].ObjectID != "CNT-CHG" {
		t.Fatalf("want the container of the call, got %+v", hld.Containers)
	}
	if len(hld.Aggregates) != 1 || hld.Aggregates[0].ID != "AGG-CHG" {
		t.Fatalf("want the aggregates of the container, got %+v", hld.Aggregates)
	}
	if len(hld.BoundedContexts) != 1 || hld.BoundedContexts[0].ID != "BC-CHG" {
		t.Fatalf("want the bounded context of the container, got %+v", hld.BoundedContexts)
	}
	if len(hld.ContextRelations) != 1 || hld.ContextRelations[0].From != "BC-CHG" {
		t.Fatalf("want the relations of the context, got %+v", hld.ContextRelations)
	}
	srs := forms["FRM-1-02"].Payload
	if len(srs.NFR) != 1 || srs.NFR[0].ID != "NFR-1" {
		t.Fatalf("want the NFR of the container, got %+v", srs.NFR)
	}
	if len(srs.FR) != 2 {
		t.Fatalf("the functional requirements have no container link, want both, got %d", len(srs.FR))
	}
	decisions := forms["FRM-1-04-ADR"].Payload.Decisions
	if len(decisions) != 1 || decisions[0].ID != "ADR-1" {
		t.Fatalf("want the decision of the container, got %+v", decisions)
	}
}

// TestFocusContextLeavesOtherStagesAlone: a stage without a branch rule keeps the
// whole context.
func TestFocusContextLeavesOtherStagesAlone(t *testing.T) {
	labels := []entryLabel{{plan: batchPlan{kind: "containers", idField: "objectId"}, idValue: "CNT-STN"}}
	got := focusContext(domain.TypeSRS, json.RawMessage(lldInputs), labels, nil)
	if string(got) != lldInputs {
		t.Fatal("a stage without a branch rule must keep the whole context")
	}
}

// TestDetailCallShipsTheBranchAndTheEntryAnswer: one contract call of the LLD
// reads the branch of its relation, gets the contract paragraphs of the registry
// page, and answers with the entry and its paths only.
func TestDetailCallShipsTheBranchAndTheEntryAnswer(t *testing.T) {
	// The registry page resolves from the module root, as it does under make run.
	t.Chdir("../..")
	var seen LLMRequest
	respond := func(request LLMRequest) (LLMResponse, error) {
		seen = request
		return LLMResponse{Payload: json.RawMessage(
			`{"entry":{"id":"CTR-CONNECTOR-STATE","kind":"sync"},` +
				`"paths":[{"id":"PATH-1","contractRef":"CTR-CONNECTOR-STATE"}]}`)}, nil
	}
	payload := json.RawMessage(`{"containers":[],"contracts":[{"id":"CTR-CONNECTOR-STATE"}],"paths":[]}`)
	stages := NewStageService(nil, nil, FixedLLM(&calledLLM{next: respond}), nil)
	merged := stages.detailLevel(
		context.Background(), StageRequest{}, &calledLLM{next: respond}, domain.AgentConfig{},
		domain.Stage{ID: "05"}, domain.TypeLLD, "", json.RawMessage(lldInputs),
		[]batchPlan{batchPlans[domain.TypeLLD][1]}, payload, "", nil, new([]json.RawMessage),
	)

	if !strings.Contains(seen.User, "Put the filled entry under the key \"entry\"") {
		t.Fatalf("the call must ask for the entry only, got:\n%s", seen.User)
	}
	if !strings.Contains(seen.User, "The entry now holds:") {
		t.Fatalf("the call must ship the entry it details, got:\n%s", seen.User)
	}
	if !strings.Contains(seen.User, "CNT-STN") || strings.Contains(seen.User, "AGG-BILL") {
		t.Fatalf("the call must ship the branch only, got:\n%s", seen.User)
	}
	if len(seen.Files) != 1 || !strings.Contains(seen.Files[0].Content, "Paragraph 3") ||
		strings.Contains(seen.Files[0].Content, "## Delta") {
		t.Fatalf("the call must ship the contract paragraphs of the page, got %d files", len(seen.Files))
	}
	if !strings.Contains(string(merged), "PATH-1") {
		t.Fatalf("the paths of the answer must merge beside the entry, got %s", merged)
	}
}

// TestEntryAnswerCarriesTheEntryOnly: the new answer shape names the entry under
// its key, and the old echo of the whole payload still merges.
func TestEntryAnswerCarriesTheEntryOnly(t *testing.T) {
	labels := []entryLabel{{plan: batchPlan{kind: "contracts"}, idValue: "CTR-STATE"}}
	entry, ok := entryFromEcho(json.RawMessage(`{"entry":{"id":"CTR-STATE","kind":"sync"}}`), labels)
	if !ok || !strings.Contains(string(entry), "CTR-STATE") {
		t.Fatalf("want the entry of the answer, got %s ok=%v", entry, ok)
	}
	legacy, ok := entryFromEcho(json.RawMessage(`{"contracts":[{"id":"CTR-STATE"}]}`), labels)
	if !ok || fieldOf(legacy, "id") != "CTR-STATE" {
		t.Fatalf("want the echo of the whole payload to merge, got %s ok=%v", legacy, ok)
	}
}

// TestRegistrySectionsKeepTheContractParagraphs: a contract call reads the
// contract, error, path paragraphs and the gates, not the whole page.
func TestRegistrySectionsKeepTheContractParagraphs(t *testing.T) {
	page := "# LLD\n\n| field | value |\n\n## Template\n\n" +
		"### Paragraph 1 - Container layout\none\n\n" +
		"### Paragraph 3 - Contract\nthree\n\n" +
		"### Paragraph 4 - Error catalog\nfour\n\n" +
		"### Paragraph 5 - Happy path\nfive\n\n" +
		"### Paragraph 6 - Negative path\nsix\n\n" +
		"### Paragraph 7 - Event map\nseven\n\n" +
		"## Gates\ngates\n\n## Delta\ndelta\n"
	focused, ok := keepSections(page, registryParagraphs[domain.TypeLLD]["contracts"])
	if !ok {
		t.Fatal("the numbered template must be found")
	}
	for _, want := range []string{"Paragraph 3", "Paragraph 4", "Paragraph 5", "Paragraph 6", "## Gates"} {
		if !strings.Contains(focused, want) {
			t.Fatalf("want %q in the focus, got:\n%s", want, focused)
		}
	}
	for _, dropped := range []string{"Paragraph 1", "Paragraph 7", "## Delta", "| field | value |"} {
		if strings.Contains(focused, dropped) {
			t.Fatalf("%q must not reach a contract call, got:\n%s", dropped, focused)
		}
	}
}

// TestRegistrySectionsKeepTheHLDContainerParagraphs: a container call reads the
// container, aggregate and invariant paragraphs and the gates, not the bounded
// context, the degradation strategy or the delta of the page.
func TestRegistrySectionsKeepTheHLDContainerParagraphs(t *testing.T) {
	page := "# HLD\n\n| field | value |\n\n## Template\n\n" +
		"### Paragraph 1 - Ubiquitous language\none\n\n" +
		"### Paragraph 2 - Bounded context\ntwo\n\n" +
		"### Paragraph 3 - Container\nthree\n\n" +
		"### Paragraph 4 - Aggregate\nfour\n\n" +
		"### Paragraph 5 - Context relation\nfive\n\n" +
		"### Paragraph 6 - Invariant (v2)\nsix\n\n" +
		"### Paragraph 7 - Degradation strategy (v2)\nseven\n\n" +
		"## Gates\ngates\n\n## Delta\ndelta\n"
	focused, ok := keepSections(page, registryParagraphs[domain.TypeHLD]["containers"])
	if !ok {
		t.Fatal("the numbered template must be found")
	}
	for _, want := range []string{"Paragraph 3", "Paragraph 4", "Paragraph 6", "## Gates"} {
		if !strings.Contains(focused, want) {
			t.Fatalf("want %q in the focus, got:\n%s", want, focused)
		}
	}
	for _, dropped := range []string{"Paragraph 1", "Paragraph 2", "Paragraph 5", "Paragraph 7", "## Delta", "| field | value |"} {
		if strings.Contains(focused, dropped) {
			t.Fatalf("%q must not reach a container call, got:\n%s", dropped, focused)
		}
	}
}

// TestDetailGapBecomesAnAnswerablePointer: a question from a detail call points
// at the entry by its id; the pointer must become an index before the gates and
// the operator queue read it.
func TestDetailGapBecomesAnAnswerablePointer(t *testing.T) {
	doc := json.RawMessage(`{
		"containers": [],
		"contracts": [{"id": "CTR-A", "kind": "sync"}],
		"paths": [],
		"openGaps": [{
			"id": "GAP-1",
			"question": {"lang": "STE100", "text": "Which input type does the contract accept?"},
			"pointer": "payload.contracts[id=CTR-A].inputType"
		}]
	}`)
	fixed := normalizeGapPointers(doc)
	if !strings.Contains(string(fixed), `"/contracts/0/inputType"`) {
		t.Fatalf("the gap must name the entry by index, got %s", fixed)
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(fixed, &probe); err != nil {
		t.Fatal(err)
	}
	if !domain.Patchable(domain.TypeLLD, fixed, "/contracts/0/inputType") {
		t.Fatal("the normalized pointer must be patchable, or the question is lost")
	}
}

// TestRegistrySectionsCutTheRealLLDPage: the page beside the module must give
// the contract paragraphs and the gates, so the mapping matches the source of
// truth and not only a fixture.
func TestRegistrySectionsCutTheRealLLDPage(t *testing.T) {
	// The test runs inside adev/internal/usecase: the registry sits three levels up.
	page, err := os.ReadFile(filepath.Join("..", "..", "..", "form-registry", "lld.md"))
	if err != nil {
		t.Skipf("the registry page is not beside the module: %v", err)
	}
	focused, ok := keepSections(string(page), registryParagraphs[domain.TypeLLD]["contracts"])
	if !ok {
		t.Fatal("the real page must carry the numbered template")
	}
	for _, want := range []string{"Paragraph 3", "Paragraph 6", "## Gates", "contract"} {
		if !strings.Contains(focused, want) {
			t.Fatalf("want %q in the focus of the real page", want)
		}
	}
	for _, dropped := range []string{"Paragraph 1", "Paragraph 7", "## Delta", "## Object use"} {
		if strings.Contains(focused, dropped) {
			t.Fatalf("%q must not reach a contract call", dropped)
		}
	}
	if len(focused) >= len(page) {
		t.Fatalf("the focus is %d of %d bytes: nothing was cut", len(focused), len(page))
	}
}

// TestRegistrySectionsFallBackToTheWholePage: a page without the numbered
// template ships whole, so a new form type never loses its page.
// TestKeepSectionsAcceptsCRLF: a Windows checkout hands the page CRLF line
// endings; the gates must still be found.
func TestKeepSectionsAcceptsCRLF(t *testing.T) {
	page := "### Paragraph 1 - Contract\r\nkeep\r\n\r\n## Gates\r\ngates\r\n"
	focused, ok := keepSections(page, []int{1})
	if !ok {
		t.Fatal("the numbered template must be found in a CRLF page")
	}
	if !strings.Contains(focused, "## Gates") {
		t.Fatalf("gates dropped from a CRLF page: %q", focused)
	}
}

func TestRegistrySectionsFallBackToTheWholePage(t *testing.T) {
	page := "# BRD\n\n## Template\n\nSome prose.\n\n## Gates\ngates\n"
	if _, ok := keepSections(page, []int{1}); ok {
		t.Fatal("a page without numbered paragraphs must report no focus")
	}
}
