package usecase

import (
	"encoding/json"
	"testing"

	"github.com/Reddetk/Adev/internal/domain"
)

// TestCoercePayload: the model answers the meaning and loses the wrapper. The
// repair touches the wrapping only: strings become the Text records, a number
// inside a sentence lands as the number, a Text keeps its language. The gates
// still judge the values the model brought.
func TestCoercePayload(t *testing.T) {
	raw := `{
		"openGaps": [
			{"blocking": false, "id": "GAP-1", "pointer": "/metrics/0/unit",
			 "question": "What unit counts one day of the calendar?"}
		],
		"metrics": [
			{"baseline": "0,5 users now", "id": "MET-1", "name": "Migrated conversations", "target": "1 user", "unit": "conversations"}
		],
		"scope": {"in": [{"text": "use emoji"}]},
		"successMetrics": [{"metricId": "MET-1", "threshold": "1 user", "window": "0 - 10 users"}]
	}`
	coerced := coercePayload(domain.TypeBRD, json.RawMessage(raw))
	var payload map[string]any
	if err := json.Unmarshal(coerced, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}

	gap := payload["openGaps"].([]any)[0].(map[string]any)
	question := gap["question"].(map[string]any)
	if question["text"] != "What unit counts one day of the calendar?" || question["lang"] != "STE100" {
		t.Fatalf("the question must land as the Text record: %v", question)
	}

	metric := payload["metrics"].([]any)[0].(map[string]any)
	if metric["baseline"] == "0,5 users now" || metric["target"] == "1 user" {
		t.Fatalf("numbers inside sentences must land as numbers: %v", metric)
	}
	for _, field := range []string{"baseline", "target"} {
		if _, ok := metric[field].(float64); !ok {
			t.Fatalf("%s must be a number, got %v", field, metric[field])
		}
	}

	in := payload["scope"].(map[string]any)["in"].([]any)[0].(map[string]any)
	if in["lang"] != "STE100" || in["text"] != "use emoji" {
		t.Fatalf("a Text without its language gets the language: %v", in)
	}

	window := payload["successMetrics"].([]any)[0].(map[string]any)["window"]
	if window != "0 - 10 users" {
		t.Fatalf("a string field keeps its words even for a number: %v", window)
	}
}

// TestNormalizeLLDLayoutDropsNestedObjects: a detail answer sometimes nests a
// container in a layer or a layer in a package. The schema must keep the outer
// entry and drop the junk, so the gates judge the real layout.
func TestNormalizeLLDLayoutDropsNestedObjects(t *testing.T) {
	raw := `{"containers":[{"containerId":"C-1","layers":[
		{"name":"domain","layers":[{"name":"domain","packages":[{"path":"internal/wallet/domain"}]}]},
		{"name":"infra","packages":[{"containerId":"C-1","layers":[{"name":"infra"}]}]}
	]}],"contracts":[],"paths":[]}`

	normalized := normalizeLLDLayout(domain.TypeLLD, json.RawMessage(raw))
	var doc domain.LLD
	if err := json.Unmarshal(normalized, &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	layers := doc.Containers[0].Layers
	if len(layers[0].Packages) != 0 {
		t.Fatalf("the nested layer must not become a package: %+v", layers[0])
	}
	if len(layers[1].Packages) != 1 || layers[1].Packages[0].Path != "" {
		t.Fatalf("the nested container must not become a package path: %+v", layers[1])
	}
}

// TestCoercePayloadMapsTheMoSCoWPriority: the model answers the MoSCoW word
// "should", and the registry enum keeps must/high/low. The synonym is the same
// decision, so the coercion maps it instead of asking the operator.
func TestCoercePayloadMapsTheMoSCoWPriority(t *testing.T) {
	raw := `{"fr":[{"id":"FR-1","priority":"should","statement":{"text":"x","priority":"should"}}]}`
	coerced := coercePayload(domain.TypeSRS, json.RawMessage(raw))
	var doc struct {
		FR []struct {
			Priority  string `json:"priority"`
			Statement struct {
				Priority string `json:"priority"`
			} `json:"statement"`
		} `json:"fr"`
	}
	if err := json.Unmarshal(coerced, &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if doc.FR[0].Priority != "high" || doc.FR[0].Statement.Priority != "high" {
		t.Fatalf("want the MoSCoW words mapped to the enum, got %+v", doc.FR[0])
	}
}

// TestCoercePayloadNormalizesTheActorFrequency: the operator says "weekly", the
// registry keeps the enum day/week/month/quarter/event.
func TestCoercePayloadNormalizesTheActorFrequency(t *testing.T) {
	raw := `{"actors":[{"id":"A-1","name":"Guest","frequency":"weekly"},{"id":"A-2","name":"Manager","frequency":"hours"}]}`
	coerced := coercePayload(domain.TypeBRD, json.RawMessage(raw))
	var doc struct {
		Actors []struct {
			Frequency string `json:"frequency"`
		} `json:"actors"`
	}
	if err := json.Unmarshal(coerced, &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if doc.Actors[0].Frequency != "week" || doc.Actors[1].Frequency != "hours" {
		t.Fatalf("want the known words mapped and the unknown one kept, got %+v", doc.Actors)
	}
}

// TestDropBRDMetricGapsKeepsTheSubjectQuestions: a new system has no measured
// value, so the questions about metrics and success values leave the draft, and a
// question about a role stays.
func TestDropBRDMetricGapsKeepsTheSubjectQuestions(t *testing.T) {
	raw := `{"goals":[],"openGaps":[
		{"id":"G1","pointer":"/metrics/0/name","question":{"text":"Which metric?","lang":"STE100"}},
		{"id":"G2","pointer":"/successMetrics/0/window","question":{"text":"Which window?","lang":"STE100"}},
		{"id":"G3","pointer":"/actors/1/name","question":{"text":"Which role?","lang":"STE100"}}]}`
	got := dropBRDMetricGaps(json.RawMessage(raw))
	var doc struct {
		OpenGaps []struct {
			ID string `json:"id"`
		} `json:"openGaps"`
	}
	if err := json.Unmarshal(got, &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(doc.OpenGaps) != 1 || doc.OpenGaps[0].ID != "G3" {
		t.Fatalf("want only the role question kept, got %+v", doc.OpenGaps)
	}
}

// TestDropFilledGapsKeepsTheUnansweredOnes: the model copies old questions into
// its new answer. A question about a field it just filled must leave, and a
// question about an empty field must stay.
func TestDropFilledGapsKeepsTheUnansweredOnes(t *testing.T) {
	raw := `{"options":[{"id":"o1","cost":{"currency":"RUB","amount":10},"confidence":""}],
		"openGaps":[
			{"id":"G1","pointer":"/options/0/cost/currency","question":{"text":"Currency?","lang":"STE100"}},
			{"id":"G2","pointer":"/options/0/confidence","question":{"text":"Confidence?","lang":"STE100"}}]}`
	got := dropFilledGaps(json.RawMessage(raw))
	var doc struct {
		OpenGaps []struct {
			ID string `json:"id"`
		} `json:"openGaps"`
	}
	if err := json.Unmarshal(got, &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(doc.OpenGaps) != 1 || doc.OpenGaps[0].ID != "G2" {
		t.Fatalf("want only the unanswered question kept, got %+v", doc.OpenGaps)
	}
}

// TestSystemOwnedFieldsAreNotAsked: the operator decides business values, not the
// ids, references and type names the merge fills. A role name is still a question.
func TestSystemOwnedFieldsAreNotAsked(t *testing.T) {
	lld := json.RawMessage(`{"contracts":[{"id":"C-1","inputType":"","paths":[]}]}`)
	gap := domain.Gap{ID: "GAP-1", Pointer: "/contracts/0/inputType", Question: domain.Text{Value: "Which type?"}}
	if askableGap(domain.TypeLLD, lld, gap) {
		t.Fatal("a type name is not an operator question")
	}
	brd := json.RawMessage(`{"actors":[{"name":""}]}`)
	role := domain.Gap{ID: "GAP-2", Pointer: "/actors/0/name", Question: domain.Text{Value: "Which role?"}}
	if !askableGap(domain.TypeBRD, brd, role) {
		t.Fatal("a role name is a business question and must stay")
	}
}

// A payload the shape does not name passes through untouched: nothing the gates
// repair alone must reach the coercion.
func TestCoercePayloadLeavesUnknownPayload(t *testing.T) {
	raw := `"a plain string"`
	if got := coercePayload(domain.TypeQuestion, json.RawMessage(raw)); string(got) != `"a plain string"` {
		t.Fatalf("a payload without gates stays as it is, got %s", got)
	}
}
