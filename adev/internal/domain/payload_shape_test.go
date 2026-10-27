package domain

import (
	"encoding/json"
	"errors"
	"testing"
)

// The skeleton replaces the hand-written strings a service used to keep. It must
// name every payload member of the form and stay valid JSON.
func TestPayloadShape(t *testing.T) {
	shape, err := PayloadShape(TypeBRD)
	if err != nil {
		t.Fatalf("brd shape: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(shape), &payload); err != nil {
		t.Fatalf("shape is not json: %v", err)
	}
	for _, member := range []string{"goals", "metrics", "scope", "actors", "constraints", "assumptions", "successMetrics", "openGaps"} {
		if _, ok := payload[member]; !ok {
			t.Fatalf("shape has no %s: %s", member, shape)
		}
	}

	goal, ok := payload["goals"].([]any)
	if !ok || len(goal) != 1 {
		t.Fatalf("goals is not a one-item list: %s", shape)
	}
	first, _ := goal[0].(map[string]any)
	outcome, _ := first["outcome"].(map[string]any)
	if outcome["text"] != "..." || outcome["lang"] != "STE100" {
		t.Fatalf("the Text shape is wrong: %s", shape)
	}
	if _, ok := first["metricId"]; !ok {
		t.Fatalf("the goal shape has no metricId: %s", shape)
	}
}

// Every form type with payload gates gets a skeleton, so a stage of that form can run.
func TestPayloadShapeCoversEveryFormWithGates(t *testing.T) {
	for _, formType := range []FormType{
		TypeUserInput, TypeBRD, TypeFeasibility, TypeSRS,
		TypeHLD, TypeADR, TypeLLD, TypeTaskPlan,
	} {
		shape, err := PayloadShape(formType)
		if err != nil {
			t.Fatalf("%s: %v", formType, err)
		}
		if !json.Valid([]byte(shape)) {
			t.Fatalf("%s: shape is not json: %s", formType, shape)
		}
	}
}

func TestPayloadShapeOfAFormWithoutGates(t *testing.T) {
	if _, err := PayloadShape(TypeQuestion); !errors.Is(err, ErrNoPayloadGates) {
		t.Fatalf("want ErrNoPayloadGates, got %v", err)
	}
}
