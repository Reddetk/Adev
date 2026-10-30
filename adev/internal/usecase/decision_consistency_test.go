package usecase

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Reddetk/Adev/internal/domain"
)

// The decisions of a greenfield design are consistent only when every one of them
// names a container of the design and a product that some container actually
// carries. These tests run the real stage walk (skeleton, one call per container,
// one call per decision, the gates, the store) and assert that consistency holds.

// hldOutline is the skeleton answer: two containers, one decision that takes the
// database that the second container carries.
const hldOutline = `{
	"boundedContexts": [
		{"id": "bc-app", "name": "Delivery", "purpose": {"lang": "STE100", "text": "Move parcels."}},
		{"id": "bc-data", "name": "Storage", "purpose": {"lang": "STE100", "text": "Keep the records."}}
	],
	"containers": [
		{"objectId": "c-app", "tech": "java", "boundedContextId": "bc-app", "purpose": {"lang": "STE100", "text": "Serve the client."}},
		{"objectId": "c-data", "tech": "postgresql", "boundedContextId": "bc-data", "purpose": {"lang": "STE100", "text": "Store the records."}}
	],
	"aggregates": [],
	"decisions": [
		{"id": "d-storage", "title": {"lang": "STE100", "text": "Storage"}, "objectRefs": ["c-data"], "techs": ["postgresql"]}
	],
	"degradationStrategy": [
		{"failure": "The database is down", "degradation": "The client sees the last status", "detection": "A health check fails", "timeoutMs": 1000}
	]
}`

// containersWithoutData is the same design without the database container: the
// decision keeps naming a product that nothing carries.
const containersWithoutData = `{
	"boundedContexts": [
		{"id": "bc-app", "name": "Delivery", "purpose": {"lang": "STE100", "text": "Move parcels."}}
	],
	"containers": [
		{"objectId": "c-app", "tech": "java", "boundedContextId": "bc-app", "purpose": {"lang": "STE100", "text": "Serve the client."}}
	],
	"aggregates": [],
	"decisions": [
		{"id": "d-storage", "title": {"lang": "STE100", "text": "Storage"}, "objectRefs": ["c-app"], "techs": ["postgresql"]}
	],
	"degradationStrategy": [
		{"failure": "The database is down", "degradation": "The client sees the last status", "detection": "A health check fails", "timeoutMs": 1000}
	]
}`

// hldResponder answers every call of the HLD walk: the outline, one container and
// the decision.
func hldResponder(outline string) func(LLMRequest) (LLMResponse, error) {
	return func(request LLMRequest) (LLMResponse, error) {
		switch {
		case strings.Contains(request.User, "the decisions entry with id"):
			return LLMResponse{Payload: json.RawMessage(`{"entry":{
				"id": "d-storage",
				"title": {"lang": "STE100", "text": "Storage"},
				"decision": {"lang": "STE100", "text": "PostgreSQL."},
				"alternatives": [{"option": {"lang": "STE100", "text": "MongoDB"}, "rejectedBecause": {"lang": "STE100", "text": "The route needs transactions."}}],
				"consequences": [{"lang": "STE100", "text": "Transactions."}],
				"objectRefs": ["c-data"],
				"techs": ["postgresql"]
			}}`)}, nil
		case strings.Contains(request.User, `the containers entry with objectId "c-data"`):
			return LLMResponse{Payload: json.RawMessage(`{"entry":{
				"objectId": "c-data", "tech": "postgresql", "boundedContextId": "bc-data",
				"purpose": {"lang": "STE100", "text": "Store the records."}
			},"aggregates":[{"id": "a-data", "root": "shipment", "moduleRef": "mod-data",
				"invariants": [{"text": "A record has one owner.", "modality": "shall", "priority": "must"}]}]}`)}, nil
		case strings.Contains(request.User, `the containers entry with objectId "c-app"`):
			return LLMResponse{Payload: json.RawMessage(`{"entry":{
				"objectId": "c-app", "tech": "java", "boundedContextId": "bc-app",
				"purpose": {"lang": "STE100", "text": "Serve the client."}
			},"aggregates":[{"id": "a-app", "root": "shipment", "moduleRef": "mod-app",
				"invariants": [{"text": "A shipment has one route.", "modality": "shall", "priority": "must"}]}]}`)}, nil
		default:
			return LLMResponse{Payload: json.RawMessage(outline)}, nil
		}
	}
}

// runHLDStage runs stage 03 of a pipeline over the fake store.
func runHLDStage(t *testing.T, respond func(LLMRequest) (LLMResponse, error)) (StageOutcome, *fakeFormStore, *FormService) {
	t.Helper()
	store := newFakeFormStore()
	forms := NewFormService(store, nil)
	client := &calledLLM{next: respond}
	stages := NewStageService(forms, nil, FixedLLM(client), nil)
	stage := domain.Stage{ID: "03", Name: "High-Level Design", Owner: domain.RoleArchitect,
		Inputs: []domain.FormType{domain.TypeSRS}, Outputs: []domain.FormType{domain.TypeHLD}}

	outcome, err := stages.runForm(
		context.Background(),
		StageRequest{PipelineID: "1", TargetSystemID: 1, Route: domain.RouteGreenfield},
		stage, domain.AgentConfig{Role: domain.RoleArchitect}, client,
		json.RawMessage(`{}`), nil, domain.TypeHLD, "FRM-1-03", 1, nil,
	)
	if err != nil {
		t.Fatalf("run form: %v", err)
	}
	return outcome, store, forms
}

// TestE2EDecisionKeepsItsContainer: a design whose decision takes PostgreSQL and
// declares a PostgreSQL container is accepted, and the decision reaches the stored
// document with the product it chose.
func TestE2EDecisionKeepsItsContainer(t *testing.T) {
	outcome, _, forms := runHLDStage(t, hldResponder(hldOutline))

	if outcome.Blocked {
		t.Fatalf("a design with a container for every product must pass, gaps: %v", outcome.Gaps)
	}
	form, err := forms.Get(context.Background(), "FRM-1-03", 1)
	if err != nil {
		t.Fatalf("read the form: %v", err)
	}
	if form.Status != domain.StatusValid {
		t.Fatalf("want a valid form, got %s", form.Status)
	}
	var hld domain.HLD
	if err := json.Unmarshal(form.Payload, &hld); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(hld.Decisions) != 1 || len(hld.Decisions[0].Techs) != 1 || hld.Decisions[0].Techs[0] != "postgresql" {
		t.Fatalf("the decision must keep its product, got %+v", hld.Decisions)
	}
	if !designCarriesTech(hld, hld.Decisions[0].Techs[0]) {
		t.Fatal("the stored design must carry the product its decision names")
	}
}

// TestE2EDecisionWithoutAContainerBlocks: the same decision without the database
// container is refused by the gate, the stage retries and then blocks, and the
// draft keeps the work instead of throwing it away.
func TestE2EDecisionWithoutAContainerBlocks(t *testing.T) {
	outcome, _, forms := runHLDStage(t, hldResponder(containersWithoutData))

	if !outcome.Blocked {
		t.Fatal("a decision that names a product no container carries must block the stage")
	}
	form, err := forms.Get(context.Background(), "FRM-1-03", 1)
	if err != nil {
		t.Fatalf("the draft must stay stored: %v", err)
	}
	if form.Status == domain.StatusValid {
		t.Fatal("the refused design must not be valid")
	}
	var hld domain.HLD
	if err := json.Unmarshal(form.Payload, &hld); err != nil {
		t.Fatalf("decode the draft: %v", err)
	}
	if len(hld.Decisions) == 0 {
		t.Fatal("the draft must keep the decision the model wrote")
	}
}

// designCarriesTech reports whether some container of the design has the tech.
func designCarriesTech(hld domain.HLD, tech string) bool {
	for _, container := range hld.Containers {
		if strings.EqualFold(container.Tech, tech) {
			return true
		}
	}
	return false
}

// TestE2ELLDSeesTheDecisionOfItsContainer: a design decision follows its
// containers, so a container call reads the choice that reaches it and not the one
// that belongs to another container.
func TestE2ELLDSeesTheDecisionOfItsContainer(t *testing.T) {
	t.Chdir("../..")
	stageContext := json.RawMessage(`{
		"FRM-1-03": {
			"formId": "FRM-1-03", "formType": "HLD", "formVersion": 1,
			"payload": {
				"boundedContexts": [
					{"id": "bc-app", "containerIds": ["c-app"]},
					{"id": "bc-bill", "containerIds": ["c-bill"]}
				],
				"containers": [
					{"objectId": "c-app", "boundedContextId": "bc-app"},
					{"objectId": "c-bill", "boundedContextId": "bc-bill"}
				],
				"decisions": [
					{"id": "d-storage", "title": {"text": "Storage"}, "decision": {"text": "PostgreSQL."}, "objectRefs": ["c-app"]},
					{"id": "d-billing", "title": {"text": "Billing"}, "decision": {"text": "A separate ledger service."}, "objectRefs": ["c-bill"]}
				],
				"aggregates": [],
				"degradationStrategy": []
			}
		}
	}`)

	var seen LLMRequest
	respond := func(request LLMRequest) (LLMResponse, error) {
		seen = request
		return LLMResponse{Payload: json.RawMessage(`{"entry":{"containerId":"c-app"}}`)}, nil
	}
	stages := NewStageService(nil, nil, FixedLLM(&calledLLM{next: respond}), nil)
	stages.detailCall(
		context.Background(), StageRequest{}, &calledLLM{next: respond}, domain.AgentConfig{},
		domain.Stage{ID: "05"}, domain.TypeLLD, "", stageContext,
		[]entryLabel{{plan: batchPlan{kind: "containers", idField: "containerId"}, idValue: "c-app"}},
		json.RawMessage(`{"containerId":"c-app"}`),
		batchPlan{kind: "containers", idField: "containerId"},
		json.RawMessage(`{"containers":[],"contracts":[],"paths":[]}`),
	)

	if !strings.Contains(seen.User, "PostgreSQL.") {
		t.Fatalf("the decision of the branch must reach the call, got:\n%s", seen.User)
	}
	if strings.Contains(seen.User, "A separate ledger service.") {
		t.Fatalf("the decision of another container must stay out, got:\n%s", seen.User)
	}
}

// TestE2ETheLLDReadsTheADRWhenTheRouteHasOne: the LLD of a change reads the ADR
// that explains the move; the LLD of a new system runs without it.
func TestE2ETheLLDReadsTheADRWhenTheRouteHasOne(t *testing.T) {
	store := newFakeFormStore()
	forms := NewFormService(store, nil)
	stages := NewStageService(forms, nil, FixedLLM(&calledLLM{}), nil)
	lld, err := domain.StageByID(domain.RouteGreenfield, "05")
	if err != nil {
		t.Fatalf("stage 05: %v", err)
	}
	request := StageRequest{PipelineID: "1", TargetSystemID: 1, Route: domain.RouteGreenfield}

	hld := &domain.Form{Envelope: domain.Envelope{
		FormID: "FRM-1-03", FormType: domain.TypeHLD, SchemaVersion: 1, FormVersion: 1,
		Status: domain.StatusValid, Route: domain.RouteGreenfield, Stage: "03", Mode: domain.ModeFull,
		PipelineID: "1", TargetSystemID: "1", ProducedBy: domain.RoleArchitect,
	}}
	if err := hld.SetRawPayload(json.RawMessage(`{"boundedContexts":[],"containers":[],"aggregates":[]}`)); err != nil {
		t.Fatalf("hld payload: %v", err)
	}
	if err := store.Save(context.Background(), hld); err != nil {
		t.Fatalf("store the hld: %v", err)
	}

	contextWithoutADR, refs, err := stages.buildContext(context.Background(), request, lld)
	if err != nil {
		t.Fatalf("the LLD must run without an ADR: %v", err)
	}
	if strings.Contains(string(contextWithoutADR), "ADR") || len(refs) != 1 {
		t.Fatalf("no ADR means one input, got %d refs and %s", len(refs), contextWithoutADR)
	}

	adr := &domain.Form{Envelope: domain.Envelope{
		FormID: "FRM-1-04-ADR", FormType: domain.TypeADR, SchemaVersion: 1, FormVersion: 1,
		Status: domain.StatusValid, Route: domain.RouteGreenfield, Stage: "04", Mode: domain.ModeFull,
		PipelineID: "1", TargetSystemID: "1", ProducedBy: domain.RoleArchitect,
	}}
	if err := adr.SetRawPayload(json.RawMessage(`{"decisions":[{"id":"d-1","title":{"text":"Storage"},"decision":{"text":"PostgreSQL."},"alternatives":[]}]}`)); err != nil {
		t.Fatalf("adr payload: %v", err)
	}
	if err := store.Save(context.Background(), adr); err != nil {
		t.Fatalf("store the adr: %v", err)
	}

	contextWithADR, refs, err := stages.buildContext(context.Background(), request, lld)
	if err != nil {
		t.Fatalf("build the context: %v", err)
	}
	if !strings.Contains(string(contextWithADR), "PostgreSQL.") || len(refs) != 2 {
		t.Fatalf("the ADR of the change must join the LLD context, got %d refs and %s", len(refs), contextWithADR)
	}
}
