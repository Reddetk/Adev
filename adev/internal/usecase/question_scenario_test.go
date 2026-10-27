package usecase

import (
	"encoding/json"
	"testing"

	"github.com/Reddetk/Adev/internal/domain"
)

// TestAnswerShapeScenario: one free-text answer to a list-format question about
// a given/when/then list fills the three fields; three sentences map in order.
func TestAnswerShapeScenario(t *testing.T) {
	one := `"The app starts the session."`
	var value any
	if err := json.Unmarshal([]byte(one), &value); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	shaped, ok := answerShape(fieldShape(domain.TypeSRS, "/fr/1/acceptance"), value)
	if !ok {
		t.Fatalf("a scenario answer must shape, got not ok")
	}
	list, ok := shaped.([]any)
	if !ok || len(list) != 1 {
		t.Fatalf("want a one-entry list, got %v", shaped)
	}
	entry, ok := list[0].(map[string]any)
	if !ok || entry["given"] != "The app starts the session" || entry["when"] != "The app starts the session" {
		t.Fatalf("the repeated sentence must fill the fields, got %v", entry)
	}

	three := `"The driver opens the map. The driver picks a station. The app starts the session."`
	var threeValue any
	if err := json.Unmarshal([]byte(three), &threeValue); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	shaped, ok = answerShape(fieldShape(domain.TypeSRS, "/fr/1/acceptance"), threeValue)
	if !ok {
		t.Fatalf("a three-sentence answer must shape, got not ok")
	}
	entry = shaped.([]any)[0].(map[string]any)
	if entry["given"] != "The driver opens the map" || entry["when"] != "The driver picks a station" || entry["then"] != "The app starts the session" {
		t.Fatalf("the sentences must map in order, got %v", entry)
	}
}
