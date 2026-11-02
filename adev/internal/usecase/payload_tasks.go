package usecase

import (
	"encoding/json"

	"github.com/Reddetk/Adev/internal/domain"
)

// fillTaskRelations writes the HLD edges the task plan touches. The model names
// the containers of each task; the system keeps only the relations with an
// endpoint among them, so TDD gets the integration premise of that task and not
// the whole context map.
func fillTaskRelations(formType domain.FormType, payload, hldPayload json.RawMessage) json.RawMessage {
	if formType != domain.TypeTaskPlan || len(hldPayload) == 0 {
		return payload
	}
	var hld domain.HLD
	if json.Unmarshal(hldPayload, &hld) != nil {
		return payload
	}
	var root map[string]json.RawMessage
	if json.Unmarshal(payload, &root) != nil {
		return payload
	}
	raw, ok := root["tasks"]
	if !ok {
		return payload
	}
	var tasks []domain.PlanTask
	if json.Unmarshal(raw, &tasks) != nil {
		return payload
	}
	relations := domain.RelationsForTasks(hld, tasks)
	encoded, err := json.Marshal(relations)
	if err != nil {
		return payload
	}
	root["contextRelations"] = encoded
	out, err := json.Marshal(root)
	if err != nil {
		return payload
	}
	return out
}
