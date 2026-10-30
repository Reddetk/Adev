package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/Reddetk/Adev/internal/domain"
)

// TestDetailOfOneLayerKeepsTheOtherLayers: the level walk replaces one layer
// inside one container entry, and keeps the other layers of that container.
func TestDetailOfOneLayerKeepsTheOtherLayers(t *testing.T) {
	payload := json.RawMessage(`{
		"containers": [
			{"containerId": "C-1", "layers": [
				{"name": "app", "packages": []},
				{"name": "infra", "packages": []}
			]}
		]
	}`)
	echo := json.RawMessage(`{
		"containers": [
			{"containerId": "C-1", "layers": [
				{"name": "app", "packages": [{"path": "internal/app", "modules": []}]},
				{"name": "infra", "packages": []}
			]}
		]
	}`)

	respond := func(request LLMRequest) (LLMResponse, error) {
		if strings.Contains(request.User, `the layers entry with name "app"`) ||
			strings.Contains(request.User, `the containers entry with containerId "C-1"`) {
			return LLMResponse{Payload: echo}, nil
		}
		return LLMResponse{}, fmt.Errorf("detail test: unexpected call %q", request.User)
	}

	stages := NewStageService(nil, nil, FixedLLM(&calledLLM{next: respond}), nil)
	merged := stages.detailLevel(
		context.Background(), StageRequest{}, &calledLLM{next: respond}, domain.AgentConfig{},
		domain.Stage{ID: "05"}, "", "{}", json.RawMessage(`{}`),
		[]batchPlan{batchPlans[domain.TypeLLD][0]}, payload, payload, "", nil, new([]json.RawMessage), nil,
	)

	var doc struct {
		Containers []struct {
			Layers []struct {
				Name     string `json:"name"`
				Packages []struct {
					Path string `json:"path"`
				} `json:"packages"`
			} `json:"layers"`
		} `json:"containers"`
	}
	if err := json.Unmarshal(merged, &doc); err != nil {
		t.Fatalf("decode merged: %v", err)
	}
	if len(doc.Containers) != 1 {
		t.Fatalf("want 1 container, got %d", len(doc.Containers))
	}
	layers := doc.Containers[0].Layers
	if len(layers) != 2 {
		t.Fatalf("want 2 layers, got %d", len(layers))
	}
	if len(layers[0].Packages) != 1 || layers[0].Packages[0].Path != "internal/app" {
		t.Fatalf("want the app layer detailed, got %+v", layers[0])
	}
	if len(layers[1].Packages) != 0 {
		t.Fatalf("the infra layer must stay unchanged, got %+v", layers[1])
	}
}

// calledLLM serves one coded response per call.
type calledLLM struct {
	next func(LLMRequest) (LLMResponse, error)
}

func (c *calledLLM) Complete(_ context.Context, request LLMRequest) (LLMResponse, error) {
	return c.next(request)
}

// TestDetailLevelSavesProgressAfterEveryEntry: a walk of an hour saves the
// document after each finished entry, so a stop or a crash keeps the answers the
// model already gave instead of throwing the run away.
func TestDetailLevelSavesProgressAfterEveryEntry(t *testing.T) {
	payload := json.RawMessage(`{"fr":[{"id":"fr-1","statement":{}},{"id":"fr-2","statement":{}}]}`)
	respond := func(request LLMRequest) (LLMResponse, error) {
		id := "fr-1"
		if strings.Contains(request.User, `"fr-2"`) {
			id = "fr-2"
		}
		return LLMResponse{Payload: json.RawMessage(
			`{"entry":{"id":"` + id + `","statement":{"text":"The client signs in."}}}`)}, nil
	}

	var saved []json.RawMessage
	checkpoint := func(document json.RawMessage) error {
		saved = append(saved, document)
		return nil
	}

	stages := NewStageService(nil, nil, FixedLLM(&calledLLM{next: respond}), nil)
	stages.detailLevel(
		context.Background(), StageRequest{}, &calledLLM{next: respond}, domain.AgentConfig{},
		domain.Stage{ID: "02"}, domain.TypeSRS, "", json.RawMessage(`{}`),
		[]batchPlan{{kind: "fr"}}, payload, payload, "", nil, new([]json.RawMessage), checkpoint,
	)

	if len(saved) != 2 {
		t.Fatalf("want one saved document per finished entry, got %d", len(saved))
	}
	var last struct {
		FR []struct {
			ID        string `json:"id"`
			Statement struct {
				Text string `json:"text"`
			} `json:"statement"`
		} `json:"fr"`
	}
	if err := json.Unmarshal(saved[len(saved)-1], &last); err != nil {
		t.Fatalf("decode the saved document: %v", err)
	}
	if len(last.FR) != 2 {
		t.Fatalf("want both entries in the saved document, got %d", len(last.FR))
	}
	for _, fr := range last.FR {
		if fr.Statement.Text == "" {
			t.Fatalf("the saved document must hold the answered entry %s", fr.ID)
		}
	}
}
