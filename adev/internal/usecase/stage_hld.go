package usecase

import (
	"context"
	"encoding/json"

	"github.com/Reddetk/Adev/internal/domain"
)

// normalizeHLDPayload applies the system-owned structure after the detail walk:
// one architecture decision, one ops deployment, and the store container the
// storage decision names. The model writes meaning; the system keeps the fork in
// the field that owns it.
func normalizeHLDPayload(formType domain.FormType, payload json.RawMessage) json.RawMessage {
	if formType != domain.TypeHLD {
		return payload
	}
	var hld domain.HLD
	if json.Unmarshal(payload, &hld) != nil {
		return payload
	}
	domain.NormalizeHLD(&hld)
	out, err := json.Marshal(hld)
	if err != nil {
		return payload
	}
	return out
}

// addExternalSystemsPayload creates the C4 external systems the BRD assumptions
// name and connects them to the internal container that uses them.
func addExternalSystemsPayload(payload json.RawMessage, brd domain.BRD, srs domain.SRS) json.RawMessage {
	var hld domain.HLD
	if json.Unmarshal(payload, &hld) != nil {
		return payload
	}
	domain.AddExternalSystems(&hld, brd, srs)
	out, err := json.Marshal(hld)
	if err != nil {
		return payload
	}
	return out
}

// hldCrossFormFailures checks the HLD against the forms it reads. The form gates
// judge the document alone; this seam judges the meaning: every BRD actor is a
// person, every edge carries an SRS requirement, and the delivery premise decides
// the edge type. A missing input is not a failure: the stage reads what the route
// produced.
func (s *StageService) hldCrossFormFailures(ctx context.Context, pipelineID string, payload json.RawMessage) []error {
	var hld domain.HLD
	if json.Unmarshal(payload, &hld) != nil {
		return nil
	}
	brd, ok := latestPayload[domain.BRD](s, ctx, pipelineID, domain.TypeBRD)
	if !ok {
		return nil
	}
	srs, ok := latestPayload[domain.SRS](s, ctx, pipelineID, domain.TypeSRS)
	if !ok {
		return nil
	}
	return domain.ValidateHLDAgainstInputs(hld, brd, srs)
}

// latestPayload decodes the newest stored version of one form type. A missing or
// broken version is an absent premise, not an error of this stage.
func latestPayload[T any](s *StageService, ctx context.Context, pipelineID string, formType domain.FormType) (T, bool) {
	var zero T
	versions, err := s.forms.ByPipeline(ctx, pipelineID, formType)
	if err != nil || len(versions) == 0 {
		return zero, false
	}
	var value T
	if json.Unmarshal(versions[len(versions)-1].Payload, &value) != nil {
		return zero, false
	}
	return value, true
}
