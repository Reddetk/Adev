package usecase

import (
	"encoding/json"

	"github.com/Reddetk/Adev/internal/domain"
)

// fillPatternFields writes the D8 checklist from the deployment pattern. The
// model chooses the pattern; the steps a pattern makes mandatory are catalogue
// knowledge, so the system writes them and the operator reads one list.
func fillPatternFields(formType domain.FormType, payload json.RawMessage) json.RawMessage {
	if formType != domain.TypeHLD {
		return payload
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(payload, &root); err != nil {
		return payload
	}
	raw, ok := root["decisions"]
	if !ok {
		return payload
	}
	var decisions []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &decisions); err != nil {
		return payload
	}
	changed := false
	for _, decision := range decisions {
		var deployment string
		if err := json.Unmarshal(decision["deployment"], &deployment); err != nil || deployment == "" {
			continue
		}
		checklist := domain.DeploymentChecklist(domain.DeploymentPattern(deployment))
		if len(checklist) == 0 {
			continue
		}
		encoded, err := json.Marshal(checklist)
		if err != nil {
			continue
		}
		decision["checklist"] = encoded
		changed = true
	}
	if !changed {
		return payload
	}
	encoded, err := json.Marshal(decisions)
	if err != nil {
		return payload
	}
	root["decisions"] = encoded
	out, err := json.Marshal(root)
	if err != nil {
		return payload
	}
	return out
}
