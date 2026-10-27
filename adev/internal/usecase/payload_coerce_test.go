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

// A payload the shape does not name passes through untouched: nothing the gates
// repair alone must reach the coercion.
func TestCoercePayloadLeavesUnknownPayload(t *testing.T) {
	raw := `"a plain string"`
	if got := coercePayload(domain.TypeQuestion, json.RawMessage(raw)); string(got) != `"a plain string"` {
		t.Fatalf("a payload without gates stays as it is, got %s", got)
	}
}
